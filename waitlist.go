package main

// todo sqlite/redis cache

type Waitlist struct {
	list map[int64]bool
}

func NewWaitlist() Waitlist {
	return Waitlist{
		list: make(map[int64]bool),
	}
}

func (w *Waitlist) AddToList(id int64) {
	w.list[id] = true
}

func (w *Waitlist) RemoveFromList(id int64) {
	delete(w.list, id)
}

func (w *Waitlist) IsExists(id int64) bool {
	_, ok := w.list[id]
	return ok
}
