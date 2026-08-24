package collector

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/matta813/pgsentinel/internal/models"
)

func (c *Core) CollectReplication(ctx context.Context) (models.ReplicationStats, error) {
	var out models.ReplicationStats
	if err := c.db.QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&out.InRecovery); err != nil {
		return out, err
	}
	if out.InRecovery {
		var receiver models.WALReceiver
		err := c.db.QueryRow(ctx, `SELECT status,COALESCE(sender_host,''),COALESCE(latest_end_lsn::text,''),COALESCE(EXTRACT(EPOCH FROM now()-last_msg_receipt_time),0) FROM pg_stat_wal_receiver LIMIT 1`).Scan(&receiver.Status, &receiver.SenderHost, &receiver.LatestEndLSN, &receiver.LastMessageSeconds)
		if err == nil {
			out.Receiver = &receiver
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, fmt.Errorf("WAL receiver: %w", err)
		}
		return out, nil
	}
	rows, err := c.db.Query(ctx, `SELECT application_name,COALESCE(client_addr::text,''),state,sync_state,COALESCE(EXTRACT(EPOCH FROM write_lag),0),COALESCE(EXTRACT(EPOCH FROM flush_lag),0),COALESCE(EXTRACT(EPOCH FROM replay_lag),0) FROM pg_stat_replication`)
	if err != nil {
		return out, fmt.Errorf("standbys: %w", err)
	}
	for rows.Next() {
		var standby models.ReplicationStandby
		if err := rows.Scan(&standby.Application, &standby.ClientAddress, &standby.State, &standby.SyncState, &standby.WriteLagSeconds, &standby.FlushLagSeconds, &standby.ReplayLagSeconds); err != nil {
			rows.Close()
			return out, err
		}
		out.Standbys = append(out.Standbys, standby)
	}
	rows.Close()
	rows, err = c.db.Query(ctx, `SELECT slot_name,slot_type,COALESCE(database,''),active,COALESCE(wal_status,''),COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(),restart_lsn),0) FROM pg_replication_slots`)
	if err != nil {
		return out, fmt.Errorf("slots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slot models.ReplicationSlot
		if err := rows.Scan(&slot.Name, &slot.Type, &slot.Database, &slot.Active, &slot.WALStatus, &slot.RetainedBytes); err != nil {
			return out, err
		}
		out.Slots = append(out.Slots, slot)
	}
	return out, rows.Err()
}

func (c *Core) CollectWAL(ctx context.Context) (models.WALStats, error) {
	var version int
	if err := c.db.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&version); err != nil {
		return models.WALStats{}, err
	}
	var out models.WALStats
	if err := c.db.QueryRow(ctx, checkpointStatsSQL(version)).Scan(&out.TimedCheckpoints, &out.RequestedCheckpoints, &out.WriteTimeMS, &out.SyncTimeMS, &out.BuffersWritten, &out.StatsReset); err != nil {
		return out, err
	}
	return out, nil
}

func checkpointStatsSQL(version int) string {
	if version >= 170000 {
		return `SELECT num_timed,num_requested,write_time,sync_time,buffers_written,stats_reset FROM pg_stat_checkpointer`
	}
	return `SELECT checkpoints_timed,checkpoints_req,checkpoint_write_time,checkpoint_sync_time,buffers_checkpoint,stats_reset FROM pg_stat_bgwriter`
}
