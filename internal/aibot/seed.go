package aibot

import "fmt"

// botColors is the rotation of avatar colours we hand out to seed bots.
// Picked to be distinct from the player default colours in [authsvc].
var botColors = []string{"#7d8da1", "#a78bfa", "#fbbf24", "#34d399", "#f472b6", "#60a5fa"}

func defaultBotEmail(idx int) string {
	return fmt.Sprintf("bot-%02d@bots.supremacy.local", idx)
}

func defaultBotDisplay(idx int) string {
	return fmt.Sprintf("bot-%02d", idx)
}

func defaultBotColor(idx int) string {
	if idx <= 0 {
		return botColors[0]
	}
	return botColors[(idx-1)%len(botColors)]
}
