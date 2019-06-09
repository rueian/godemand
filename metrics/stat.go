package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	MResourceCount = stats.Int64("godemand/resource/count", "The current number of resources", "1")
	MResourceLife  = stats.Float64("godemand/resource/life", "The lifetime of resources", "s")
	MClientCount   = stats.Int64("godemand/client/count", "The current number of clients", "1")
	MClientLife    = stats.Float64("godemand/client/life", "The lifetime of clients", "s")
	MClientWait    = stats.Float64("godemand/client/wait", "The lifetime of clients", "s")

	KeyPool, _     = tag.NewKey("pool")
	KeyState, _    = tag.NewKey("state")
	KeyResource, _ = tag.NewKey("resource")
	KeyClient, _   = tag.NewKey("client")

	ResourceCountView = &view.View{
		Name:        "godemand/resource/count",
		Measure:     MResourceCount,
		Description: "The current number of resources",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyPool, KeyState},
	}

	ResourceLifeView = &view.View{
		Name:        "godemand/resource/life",
		Measure:     MResourceLife,
		Description: "The lifetime of resources",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyPool, KeyResource, KeyState},
	}

	ClientCountView = &view.View{
		Name:        "godemand/client/count",
		Measure:     MClientCount,
		Description: "The current number of clients",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyPool},
	}

	ClientLifeView = &view.View{
		Name:        "godemand/client/life",
		Measure:     MClientLife,
		Description: "The lifetime of clients",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyPool, KeyClient},
	}

	ClientWaitView = &view.View{
		Name:        "godemand/client/wait",
		Measure:     MClientWait,
		Description: "The waiting time of clients",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyPool, KeyClient},
	}
)

func StartRecording(period time.Duration, es ...view.Exporter) error {
	for _, e := range es {
		view.RegisterExporter(e)
	}
	view.SetReportingPeriod(period)

	for {
		if err := view.Register(ResourceCountView, ResourceLifeView, ClientCountView, ClientLifeView, ClientWaitView); err != nil {
			return err
		}
		// since we trace each resource and client, it is necessary unregistering views periodically to avoid memory leak.
		time.Sleep(period * 3)
		view.Unregister(ResourceCountView, ResourceLifeView, ClientCountView, ClientLifeView, ClientWaitView)
	}
}

func RecordResourceCount(pool, state string, count int64) {
	ctx, _ := tag.New(
		context.Background(),
		tag.Insert(KeyPool, pool),
		tag.Insert(KeyState, state),
	)

	stats.Record(ctx, MResourceCount.M(count))
}

func RecordResourceLife(pool, state, resource string, duration time.Duration) {
	ctx, _ := tag.New(
		context.Background(),
		tag.Insert(KeyPool, pool),
		tag.Insert(KeyResource, resource),
		tag.Insert(KeyState, state),
	)

	stats.Record(ctx, MResourceLife.M(duration.Seconds()))
}

func RecordClientCount(pool string, count int64) {
	ctx, _ := tag.New(
		context.Background(),
		tag.Insert(KeyPool, pool),
	)

	stats.Record(ctx, MClientCount.M(count))
}

func RecordClientLife(pool, client string, duration time.Duration) {
	ctx, _ := tag.New(
		context.Background(),
		tag.Insert(KeyPool, pool),
		tag.Insert(KeyClient, client),
	)

	stats.Record(ctx, MClientLife.M(duration.Seconds()))
}

func RecordClientWait(pool, client string, duration time.Duration) {
	ctx, _ := tag.New(
		context.Background(),
		tag.Insert(KeyPool, pool),
		tag.Insert(KeyClient, client),
	)

	stats.Record(ctx, MClientWait.M(duration.Seconds()))
}
