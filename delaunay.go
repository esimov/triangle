package triangulator

type Point struct {
	x, y int
}

type Node struct {
	X, Y int
}

type circle struct {
	x, y, radius int
}

func newNode(x, y int) Node {
	return Node{x, y}
}

func (n Node) isEq(p Node) bool {
	dx := n.X - p.X
	dy := n.Y - p.Y

	if  dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if float64(dx) < 0.0001 && float64(dy) < 0.0001 {
		return true
	}
	return false
}

type edge struct {
	nodes []Node
}

func newEdge(p0, p1 Node) []Node {
	nodes := []Node{p0, p1}
	return nodes
}

func (e edge) isEq(edge edge) bool {
	na := e.nodes
	nb := edge.nodes
	na0, na1 := na[0], na[1]
	nb0, nb1 := nb[0], nb[1]

	if (na0.isEq(nb0) && na1.isEq(nb1)) ||
		(na0.isEq(nb1) && na1.isEq(nb0)) {
		return true
	}
	return false
}

type Triangle struct {
	Nodes  []Node
	edges  []edge
	circle circle
}

var t Triangle = Triangle{}

func (t Triangle) newTriangle(p0, p1, p2 Node) Triangle {
	t.Nodes = []Node{p0, p1, p2}
	t.edges = []edge{edge{newEdge(p0, p1)}, edge{newEdge(p1, p2)}, edge{newEdge(p2, p0)}}

	circle := t.circle
	ax, ay := p1.X - p0.X, p1.Y - p0.Y
	bx, by := p2.X - p0.X, p2.Y - p0.Y

	m := p1.X * p1.X - p0.X * p0.X + p1.Y * p1.Y - p0.Y * p0.Y
	u := p2.X * p2.X - p0.X * p0.X + p2.Y * p2.Y - p0.Y * p0.Y
	s := 1.0 / (2.0 * (float64(ax * by) - float64(ay * bx)))

	circle.x = int(float64((p2.Y - p0.Y) * m + (p0.Y - p1.Y) * u) * s)
	circle.y = int(float64((p0.X - p2.X) * m + (p1.X - p0.X) * u) * s)

	dx := p0.X - circle.x
	dy := p0.Y - circle.y

	circle.radius = dx * dx + dy * dy
	t.circle = circle

	return t
}

type Delaunay struct{
	width int
	height int
	triangles []Triangle
}

func (d *Delaunay) Init(width, height int) *Delaunay {
	d.width = width
	d.height = height

	d.triangles = nil
	d.clear()

	return d
}

func (d *Delaunay) clear() {
	p0 := newNode(0, 0)
	p1 := newNode(d.width, 0)
	p2 := newNode(d.width, d.height)
	p3 := newNode(0, d.height)

	// Create the supertriangle
	d.triangles = []Triangle{t.newTriangle(p0, p1, p2), t.newTriangle(p0, p2, p3)}
}

func (d *Delaunay) Insert(points []Point) *Delaunay {
	var (
		i, j, k int
		x, y, dx, dy, distSq int
		polygon []edge
		edges []edge = []edge{}
		temps []Triangle = []Triangle{}
	)

	for k = 0; k < len(points); k++ {
		x = points[k].x
		y = points[k].y

		triangles := d.triangles
		edges = nil
		temps = nil

		for i = 0; i < len(d.triangles); i++ {
			t := triangles[i]

			circle := t.circle
			dx = circle.x - x
			dy = circle.y - y
			distSq = dx * dx + dy * dy

			if distSq < circle.radius {
				edges = append(edges, t.edges[0], t.edges[1], t.edges[2])
			} else {
				temps = append(temps, t)
			}
		}

		polygon = nil
		edgesLoop:
		for i = 0; i < len(edges); i++ {
			edge := edges[i]
			for j = 0; j < len(polygon); j++ {
				if edge.isEq(polygon[j]) {
					// Remove polygon from the polygon slice
					polygon = append(polygon[:j], polygon[j+1:]...)
					continue edgesLoop
				}
			}
			// Insert new edge into the polygon slice
			polygon = append(polygon, edge)

		}
		for i = 0; i < len(polygon); i++ {
			edge := polygon[i]
			temps = append(temps, t.newTriangle(edge.nodes[0], edge.nodes[1], newNode(x, y)))
		}
		d.triangles = temps
	}
	return d
}

func (d *Delaunay) GetTriangles() []Triangle {
	return d.triangles
}
