package grpc

import (
	"fmt"
	"math"
	"time"

	"github.com/reveald/reveald"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func mapRequest(in *Request) *reveald.Request {
	out := reveald.NewRequest()

	for _, p := range in.Parameters {
		out.Append(reveald.NewParameter(p.GetName(), p.GetValues()...))
	}

	return out
}

func mapResponse(r *reveald.Result, conv func(map[string]interface{}) (proto.Message, bool)) *Result {
	out := &Result{}
	out.TotalHitCount = r.TotalHitCount
	out.Duration = int64(r.Duration / time.Millisecond)
	out.Hits = buildHits(r.Hits, conv)
	out.Buckets = buildBuckets(r.Aggregations)
	out.Pages = buildPagination(r.Pagination, r.TotalHitCount)
	out.Sort = buildSort(r.Sorting)

	return out
}

func buildHits(raw []map[string]interface{}, conv func(map[string]interface{}) (proto.Message, bool)) []*anypb.Any {
	var hits []*anypb.Any

	for _, hit := range raw {
		msg, ok := conv(hit)
		if !ok {
			continue
		}

		s, err := proto.Marshal(msg)
		if err != nil {
			continue
		}

		a := &anypb.Any{
			TypeUrl: "reveald.hit",
			Value:   s,
		}
		hits = append(hits, a)
	}

	return hits
}

func buildBuckets(raw map[string][]*reveald.ResultBucket) map[string]*BucketList {
	buckets := make(map[string]*BucketList)

	for k, rb := range raw {
		bl := &BucketList{}
		for _, v := range rb {
			b := &Bucket{}
			b.HitCount = v.HitCount
			b.Value = fmt.Sprintf("%v", v.Value)
			bl.Values = append(bl.Values, b)
		}
		buckets[k] = bl
	}

	return buckets
}

func buildPagination(raw *reveald.ResultPagination, hc int64) *PageResult {
	if raw == nil {
		return nil
	}

	pr := &PageResult{}
	count := float64(hc) / float64(raw.PageSize)
	if math.Mod(count, 1) != 0 {
		count = count + 1
	}
	pr.Count = int64(math.Floor(count))

	current := 1
	if raw.Offset > 0 {
		diff := raw.Offset / raw.PageSize
		current = diff + 1
	}
	pr.Current = int64(current)

	return pr
}

func buildSort(raw *reveald.ResultSorting) []*SortOption {
	if raw == nil {
		return nil
	}

	var list []*SortOption

	for _, o := range raw.Options {
		so := &SortOption{}
		so.Ascending = o.Ascending
		so.Name = o.Name
		so.Selected = o.Selected

		list = append(list, so)
	}

	return list
}
