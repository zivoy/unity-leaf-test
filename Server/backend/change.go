package backend

type Change interface {
	dummy()
}

type UpdateEntityChange struct {
	baseEvent
	Entity *Entity
}

func (c *UpdateEntityChange) dummy() {}

type RemoveEntityChange struct {
	baseEvent
}

func (c *RemoveEntityChange) dummy() {}

type AddEntityChange struct {
	baseEvent
	Entity *Entity
}

func (c *AddEntityChange) dummy() {}
