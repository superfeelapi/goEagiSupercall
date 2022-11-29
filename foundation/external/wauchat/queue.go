package wauchat

import "errors"

type Queue struct {
	data []Result
}

func NewQueue() *Queue {
	return &Queue{data: []Result{}}
}

func (q *Queue) Dequeue() (Result, error) {
	if len(q.data) < 1 {
		return Result{}, errors.New("wauchat emotion queue is empty")
	}
	get := q.data[0]
	q.data = q.data[1:]
	return get, nil
}

func (q *Queue) Enqueue(d Result) {
	q.data = append(q.data, d)
}
