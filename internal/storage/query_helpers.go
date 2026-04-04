package storage

import (
	"encoding/json"

	"gorm.io/gorm"
)

func applyPagination(tx *gorm.DB, limit, offset int) *gorm.DB {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return tx.Limit(limit).Offset(offset)
}

func summaryRowsQuery(tx *gorm.DB) *gorm.DB {
	return tx.Table("summaries AS s").
		Select("s.id, s.item_id, i.title AS item_title, i.url AS item_url, s.summary, s.key_points, s.created_at").
		Joins("LEFT JOIN items i ON s.item_id = i.id")
}

type pipelineGraphRefSnapshot struct {
	GroupRefs []struct {
		GroupID        int64 `json:"group_id"`
		GroupVersionID int64 `json:"group_version_id"`
	} `json:"group_refs"`
}

func parsePipelineGraphRefs(graphJSON string) (pipelineGraphRefSnapshot, error) {
	var out pipelineGraphRefSnapshot
	err := json.Unmarshal([]byte(graphJSON), &out)
	return out, err
}

func countGroupRefsFromGraphs(defs []PipelineDefinition, groupID int64, currentVersionID *int64) (int, int) {
	var refs, outdated int
	for _, def := range defs {
		graph, err := parsePipelineGraphRefs(def.GraphJSON)
		if err != nil {
			continue
		}
		for _, ref := range graph.GroupRefs {
			if ref.GroupID != groupID {
				continue
			}
			refs++
			if currentVersionID != nil && ref.GroupVersionID != *currentVersionID {
				outdated++
			}
		}
	}
	return refs, outdated
}
