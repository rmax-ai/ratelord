package graph

// NodeType represents the semantic type of a node in the constraint graph.
type NodeType string

const (
	NodeProvider   NodeType = "provider"
	NodeIdentity   NodeType = "identity"
	NodeScope      NodeType = "scope"
	NodePool       NodeType = "pool"
	NodeConstraint NodeType = "constraint"
	NodeWorkload   NodeType = "workload"
	NodeResource   NodeType = "resource"
)

// EdgeType represents the semantic relationship between two nodes.
type EdgeType string

const (
	EdgeObserves     EdgeType = "observes"      // Provider -> Constraint
	EdgeAppliesTo    EdgeType = "applies_to"    // Constraint -> Scope
	EdgeCharges      EdgeType = "charges"       // Usage -> Identity (Conceptual, maybe not in static graph)
	EdgeConsumesFrom EdgeType = "consumes_from" // Identity/Scope -> Pool
	EdgeBounds       EdgeType = "bounds"        // Pool -> Constraint
	EdgeSharedWith   EdgeType = "shared_with"   // Pool -> Identity
	EdgeOwns         EdgeType = "owns"          // Identity -> Resource (e.g. Org owns Repo)
	EdgeTriggers     EdgeType = "triggers"      // Workload -> Identity
	EdgeLimits       EdgeType = "limits"        // Constraint -> Workload/Identity
)

// Node represents a vertex in the constraint graph.
type Node struct {
	ID         string            `json:"id"`
	Type       NodeType          `json:"type"`
	Label      string            `json:"label"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	FromID string   `json:"from_id"`
	ToID   string   `json:"to_id"`
	Type   EdgeType `json:"type"`
}

// Graph represents the canonical constraint graph snapshot.
type Graph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`
}

// NewGraph creates an empty constraint graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(n *Node) {
	g.Nodes[n.ID] = n
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(e *Edge) {
	g.Edges = append(g.Edges, e)
}
