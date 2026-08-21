package domain

import "context"

type Repository interface {
	Write(ctx context.Context, points []Point) error
	Query(ctx context.Context, query Query) ([]Point, error)
	Aggregate(ctx context.Context, request AggregateRequest) ([]AggregateResult, error)
	DetectGaps(ctx context.Context, request GapRequest) ([]Gap, error)
}
