package authsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/auth"
	"github.com/joaquing/clone-supremacy/internal/domain/userdom"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// UsersRepository is the dependency the auth service needs. The Postgres
// implementation lives in [pgrepo.Users].
type UsersRepository interface {
	Create(ctx context.Context, email, hash, display, color string) (userdom.User, error)
	ByEmail(ctx context.Context, email string) (userdom.User, error)
	ByID(ctx context.Context, id uuid.UUID) (userdom.User, error)
	TouchLogin(ctx context.Context, id uuid.UUID) error
	ListBots(ctx context.Context, limit int) ([]userdom.User, error)
}

// Service implements the registration/login/ws-ticket flows.
type Service struct {
	users   UsersRepository
	issuer  *auth.Issuer
	tickets *auth.TicketBroker
}

func New(users UsersRepository, issuer *auth.Issuer, tickets *auth.TicketBroker) *Service {
	return &Service{users: users, issuer: issuer, tickets: tickets}
}

// RegisterInput is what the controller passes after parsing the request.
type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
}

// LoginInput is the credential payload.
type LoginInput struct {
	Email    string
	Password string
}

// TokenIssued is the token+expiry pair returned by Register and Login.
type TokenIssued struct {
	AccessToken string
	ExpiresAt   time.Time
	User        userdom.User
}

// Register creates a new account, hashes the password and issues a
// session token.
func (s *Service) Register(ctx context.Context, in RegisterInput) (TokenIssued, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || in.DisplayName == "" {
		return TokenIssued{}, errs.New(errs.BadRequest, "email and display_name are required")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return TokenIssued{}, errs.Wrap(err, errs.BadRequest, "weak password")
	}
	u, err := s.users.Create(ctx, email, hash, in.DisplayName, defaultColor(email))
	if err != nil {
		if errors.Is(err, userdom.ErrExists) {
			return TokenIssued{}, errs.New(errs.Conflict, "email is already registered")
		}
		return TokenIssued{}, errs.Wrap(err, errs.Internal, "creating user")
	}
	return s.issue(u)
}

// Login verifies credentials and issues a session token.
func (s *Service) Login(ctx context.Context, in LoginInput) (TokenIssued, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return TokenIssued{}, errs.New(errs.Unauthorized, "email or password incorrect")
	}
	if u.IsBot {
		return TokenIssued{}, errs.New(errs.Unauthorized, "bot accounts cannot use password login")
	}
	if err := auth.VerifyPassword(u.PasswordHash, in.Password); err != nil {
		return TokenIssued{}, errs.New(errs.Unauthorized, "email or password incorrect")
	}
	_ = s.users.TouchLogin(ctx, u.ID)
	return s.issue(u)
}

// LoginAsBot issues a session token for a bot service account. The
// caller has already proven it is the ai-bot subsystem (via the
// pre-shared BOT_API_KEY checked at the controller boundary), so we
// don't need a password here — only a check that the user is actually
// flagged as a bot.
func (s *Service) LoginAsBot(ctx context.Context, userID uuid.UUID) (TokenIssued, error) {
	u, err := s.users.ByID(ctx, userID)
	if err != nil {
		return TokenIssued{}, errs.New(errs.NotFound, "bot account not found")
	}
	if !u.IsBot {
		return TokenIssued{}, errs.New(errs.Forbidden, "user is not a bot")
	}
	_ = s.users.TouchLogin(ctx, u.ID)
	return s.issue(u)
}

// ListBots returns the seed pool of bot identities the ai-bot
// subsystem can rotate through. Limit is hard-coded to a sensible
// upper bound because the caller is internal.
func (s *Service) ListBots(ctx context.Context) ([]userdom.User, error) {
	bots, err := s.users.ListBots(ctx, 32)
	if err != nil {
		return nil, errs.Wrap(err, errs.Internal, "listing bots")
	}
	return bots, nil
}

// MintWSTicket exchanges a session for a single-use ticket the gateway
// will redeem at WebSocket upgrade time.
func (s *Service) MintWSTicket(ctx context.Context, claims *auth.Claims) (string, error) {
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", errs.Wrap(err, errs.BadRequest, "bad subject")
	}
	t, err := s.tickets.Mint(ctx, auth.Ticket{
		UserID: uid, Email: claims.Email, DisplayName: claims.DisplayName,
	})
	if err != nil {
		return "", errs.Wrap(err, errs.Internal, "minting ticket")
	}
	return t, nil
}

func (s *Service) issue(u userdom.User) (TokenIssued, error) {
	tok, expiry, err := s.issuer.IssueAccess(auth.Identity{
		UserID: u.ID, Email: u.Email, DisplayName: u.DisplayName,
	})
	if err != nil {
		return TokenIssued{}, errs.Wrap(err, errs.Internal, "issuing token")
	}
	return TokenIssued{AccessToken: tok, ExpiresAt: expiry, User: u}, nil
}

// defaultColor produces a stable per-email default avatar colour. Lives
// here so the controller doesn't have to reach into a util package.
func defaultColor(email string) string {
	if len(email) == 0 {
		return "#5fa8ff"
	}
	colors := []string{"#e8523a", "#3a8ce8", "#3ae888", "#e8c93a", "#a23ae8", "#e83a8e"}
	return colors[int(email[0])%len(colors)]
}
