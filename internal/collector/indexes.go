package collector

import (
	"context"
	"github.com/matta813/pgsentinel/internal/models"
)

func (c *Core) CollectIndexes(ctx context.Context) ([]models.IndexStat, error) {
	rows, err := c.db.Query(ctx, `SELECT current_database(),n.nspname,t.relname,i.relname,pg_get_indexdef(i.oid),pg_relation_size(i.oid),COALESCE(s.idx_scan,0),x.indisunique,x.indisprimary FROM pg_index x JOIN pg_class i ON i.oid=x.indexrelid JOIN pg_class t ON t.oid=x.indrelid JOIN pg_namespace n ON n.oid=t.relnamespace LEFT JOIN pg_stat_user_indexes s ON s.indexrelid=i.oid WHERE n.nspname NOT IN ('pg_catalog','information_schema') ORDER BY pg_relation_size(i.oid) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.IndexStat{}
	for rows.Next() {
		var i models.IndexStat
		if err := rows.Scan(&i.Database, &i.Schema, &i.Table, &i.Index, &i.Definition, &i.SizeBytes, &i.Scans, &i.Unique, &i.Primary); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
