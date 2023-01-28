package backend

type Change interface {
}

type UpdateEntityChange struct {
	Change
	baseEvent
	Entity *Entity
}

type RemoveEntityChange struct {
	Change
	baseEvent
}

type AddEntityChange struct {
	Change
	baseEvent
	Entity *Entity
}
