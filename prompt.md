You are a Senior Gameplay Programmer and Systems Architect specializing in RTS/Grand Strategy mechanics. Your task is to implement a robust, real-time unit movement and pathfinding system modeled after "Supremacy 1914". You have full access to the repository. Please design and code this feature with performance and state synchronization in mind.

---

### 1. Core Movement & Pathfinding Engine (Node & Edge Targeting)
- **Graph Generation:** Treat the SVG map provinces as nodes and the travel routes connecting them as weighted edges. 
- **Arbitrary Edge Targeting:** Crucially, a destination target does *not* have to be a province center (node). A player can target any point along a connecting route (edge), and the unit must treat that exact sub-coordinate or percentage point along the edge as its destination.
- **Continuous Positioning:** Units must traverse along the graph edges over time. The application must track their exact coordinates/progress percentage between nodes so they can exist, be selected, halt, or change directions while mid-route.

### 2. Drag-and-Drop Interaction & Waypoint Queuing
- **Drag-to-Move Gesture:** Issuing a movement command must be handled via dragging. Clicking and holding a unit token initiates the drag state. Releasing the mouse/touch gesture (dropping) commits the movement command and starts the unit's journey to that location.
- **Multi-Stop Queuing:** Allow players to queue multiple movements. While holding a modifier key (like Shift), dragging and dropping the unit should append a new "stop" or waypoint to its existing path instead of overwriting it. Waypoints can be placed at province centers or arbitrary points along an edge.

### 3. Combat Initiation Logic
- **Diplomatic/State Checks:** Intersect the movement logic with the province ownership state. 
- **Attack Trigger:** If a unit’s dropped destination or path takes it into a province designated as hostile/enemy territory, entering that node or engaging a hostile unit along an edge must halt movement and trigger the combat/attack state machine.

### 4. UI/UX: Drag Previews & Path Vectoring
- **Real-Time Drag Preview:** While the user is actively dragging a unit token across the map, calculate the pathfinding on-the-fly from the unit's current location to the nearest valid graph position (province center or edge line) underneath the cursor. 
- **Live Vector Arrow:** Render a dynamic path arrow overlay in real-time *during* the drag state to preview the exact route the unit will take. If the player drops the unit, this preview path becomes the actual active path.
- **Static Path Display:** When a player simply selects an already moving unit, the frontend must render a visual line/arrow overlay mapping out its active trajectory and all queued waypoints.
- **Visual Specifications:** - Trace paths cleanly from the unit's exact current position to the final target.
  - Use clear visual coding (e.g., dotted lines for queued paths, solid arrows for active legs, and color shifts like red if the path terminates in an attack state).
  - Ensure these vectors scale and pan flawlessly with the SVG map.

---

### Deliverables Required:
1. **State & Interaction Architecture:** Outline how the drag-and-drop lifecycle (DragStart, DragOver/CalculatePath, DragEnd/CommitMove) maps to your state management layer.
2. **Pathfinding & Drag-Snapping Logic:** Implement the core pathfinding script that dynamically snaps the cursor to the nearest node or edge during a drag event to draw the preview.
3. **Frontend Overlay Implementation:** Provide the React/Vue/Svelte or Canvas component responsible for rendering the live directional path arrows both during a drag preview and when a moving unit is selected.