package triangulator

import "fmt"

type point struct {
	x, y, id int
}

type Node struct {
	X, Y int
}

type circle struct {
	x, y, radius int
}

var nodes []Node = make([]Node, 0)

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
	nodes = nil
	nodes = append(nodes,  p0, p1)
	return nodes
}

func (e edge) isEq(edge edge) bool {
	na := e.nodes
	nb := edge.nodes
	na0, na1 := na[0], na[1]
	nb0, nb1 := nb[0], nb[1]

	if na0.isEq(nb0) && na1.isEq(nb1) ||
		na0.isEq(nb1) && na1.isEq(nb0) {
		return true
	}
	return false
}

type Triangle struct {
	Nodes  []Node
	Edges  []edge
	circle circle
}

var t Triangle

func (t Triangle) newTriangle(p0, p1, p2 Node) Triangle {
	t.Nodes = nil
	t.Edges = nil

	t.Nodes = append(t.Nodes, p0, p1, p2)
	t.Edges = append(t.Edges, edge{newEdge(p0, p1)}, edge{newEdge(p1, p2)}, edge{newEdge(p2, p0)})
	//fmt.Println("Nodes: ", t.nodes)
	//fmt.Println("Edges: ", t.edges)
	//fmt.Println("==================")
	circle := t.circle

	ax, ay := p1.X - p0.X, p1.Y - p0.Y
	bx, by := p2.X - p0.X, p2.Y - p0.Y
	m := p1.X * p1.X - p0.X * p0.X + p1.Y * p1.Y - p0.Y * p0.Y
	u := p2.X * p2.X - p0.X * p0.X + p2.Y * p2.Y - p0.Y * p0.Y
	//fmt.Println("ax:", ax, ":", "bx:", ay)
	//fmt.Println("bx:", bx, ":", "bx:", by)
	s := 1.0 / (2.0 * float64(ax * by) - float64(ay * bx))
	circle.x = int(float64((p2.Y - p0.Y) * m + (p0.Y - p1.Y) * u) * s)
	circle.y = int(float64((p0.X - p2.X) * m + (p1.X - p0.X) * u) * s)
	//fmt.Println("s:", int(float64((p2.y - p0.y) * m + (p0.y - p1.y) * u) * s))
	//fmt.Println("circle: ", circle.x, ":", circle.y)
	dx := p0.X - circle.x
	dy := p0.Y - circle.y
	//fmt.Println("dx: ", dx)
	circle.radius = dx * dx + dy * dy
	t.circle = circle
	//fmt.Println("radius: ", circle.radius)
	//fmt.Println("nodes: ", t.nodes)
	//fmt.Println("edges: ", t.edges)
	return t
}

type Delaunay struct{
	width int
	height int
	triangles []Triangle
}

func (d *Delaunay) Init(width, height int) *Delaunay {
	t = Triangle{}

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

	d.triangles = nil
	d.triangles = append(d.triangles, t.newTriangle(p0, p2, p3), t.newTriangle(p0, p1, p2))
	//fmt.Println("Triangles:", d.triangles)
	//fmt.Println("Triangles length:", len(d.triangles))
}

func (d *Delaunay) Insert(points []point) *Delaunay {
	var (
		k, i int
		x, y, dx, dy, distSq int
		polygon []edge
	)

	fmt.Println(len(points))
	for k = 0; k < len(points); k++ {
		x = points[k].x
		y = points[k].y

		edges := []edge{}
		temps := []Triangle{}
		//fmt.Println(temps)
		for i = 0; i < len(d.triangles); i++ {
			t := d.triangles[i]

			circle := t.circle
			dx = circle.x - x
			dy = circle.y - y
			distSq = dx * dx + dy * dy

			if distSq < circle.radius {
				edges = append(edges, t.Edges[0], t.Edges[1], t.Edges[2])
			} else {
				temps = append(temps, t)
			}
		}

		//fmt.Println(edges)
		polygon = nil
		for i = 0; i < len(edges); i++ {
			edgesLoop:
			for i = 0; i < len(edges); i++ {
				edge := edges[i]
				//fmt.Println("Len:", len(polygon))
				for j := 0; j < len(polygon); j++ {
					if edge.isEq(polygon[j]) {
						//fmt.Println(edg)
						//fmt.Println(polygon[j])
						polygon = append(polygon[:j], polygon[j+1:]...)
						//fmt.Println("After: ", polygon)
						continue edgesLoop
					}
				}
				polygon = append(polygon, edge)
				//fmt.Println(polygon)
			}
		}
		//fmt.Println(len(polygon))
		for i = 0; i < len(polygon); i++ {
			edge := polygon[i]
			//node := newNode(x, y)
			//fmt.Println(node)
			//fmt.Println(edge.nodes[0])
			temps = append(temps, t.newTriangle(edge.nodes[0], edge.nodes[1], newNode(x, y)))
		}
		//fmt.Println(temps)
		//fmt.Println("Polygon length:", len(polygon))
		d.triangles = temps
	}
	//fmt.Println("+=============+")
	//fmt.Println(d.triangles)
	return d
}

func (d *Delaunay) GetTriangles() []Triangle {
	return d.triangles
}
