package collector

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

func (c *Core) CollectReplication(ctx context.Context) (models.ReplicationStats, error) {
	out := models.ReplicationStats{CollectedAt: time.Now().UTC()}
	var version int
	if err := c.db.QueryRow(ctx, `SELECT pg_is_in_recovery(),current_setting('server_version_num')::int`).Scan(&out.InRecovery, &version); err != nil {
		return out, err
	}
	_ = c.db.QueryRow(ctx, `SELECT (pg_control_checkpoint()).timeline_id`).Scan(&out.TimelineID)
	if out.InRecovery {
		if err := c.db.QueryRow(ctx, `SELECT pg_is_wal_replay_paused(),COALESCE(pg_last_wal_replay_lsn()::text,''),COALESCE(pg_last_wal_receive_lsn()::text,''),(SELECT setting::float8/1000 FROM pg_settings WHERE name='recovery_min_apply_delay')`).Scan(&out.RecoveryPaused, &out.ReplayLSN, &out.ReceiveLSN, &out.ReplayDelaySeconds); err != nil {
			return out, fmt.Errorf("recovery state: %w", err)
		}
		var receiver models.WALReceiver
		err := c.db.QueryRow(ctx, `SELECT status,COALESCE(sender_host,''),COALESCE(latest_end_lsn::text,''),COALESCE(EXTRACT(EPOCH FROM now()-last_msg_receipt_time),0) FROM pg_stat_wal_receiver LIMIT 1`).Scan(&receiver.Status, &receiver.SenderHost, &receiver.LatestEndLSN, &receiver.LastMessageSeconds)
		if err == nil {
			out.Receiver = &receiver
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, fmt.Errorf("WAL receiver: %w", err)
		}
		return out, nil
	}
	rows, err := c.db.Query(ctx, standbyStatsSQL())
	if err != nil {
		return out, fmt.Errorf("standbys: %w", err)
	}
	for rows.Next() {
		var s models.ReplicationStandby
		if err := rows.Scan(&s.Application, &s.ClientAddress, &s.State, &s.SyncState, &s.WriteLagSeconds, &s.FlushLagSeconds, &s.ReplayLagSeconds, &s.SentLSN, &s.WriteLSN, &s.FlushLSN, &s.ReplayLSN, &s.PendingSendBytes, &s.PendingWriteBytes, &s.PendingFlushBytes, &s.PendingReplayBytes, &s.ReplyAgeSeconds); err != nil {
			rows.Close()
			return out, err
		}
		out.Standbys = append(out.Standbys, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows, err = c.db.Query(ctx, replicationSlotsSQL(version))
	if err != nil {
		return out, fmt.Errorf("slots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s models.ReplicationSlot
		if err := rows.Scan(&s.Name, &s.Type, &s.Database, &s.Active, &s.WALStatus, &s.RetainedBytes, &s.InactiveSeconds); err != nil {
			return out, err
		}
		out.Slots = append(out.Slots, s)
	}
	return out, rows.Err()
}
func standbyStatsSQL() string {
	return `SELECT application_name,COALESCE(client_addr::text,''),state,sync_state,COALESCE(EXTRACT(EPOCH FROM write_lag),0),COALESCE(EXTRACT(EPOCH FROM flush_lag),0),COALESCE(EXTRACT(EPOCH FROM replay_lag),0),COALESCE(sent_lsn::text,''),COALESCE(write_lsn::text,''),COALESCE(flush_lsn::text,''),COALESCE(replay_lsn::text,''),COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(),sent_lsn),0),COALESCE(pg_wal_lsn_diff(sent_lsn,write_lsn),0),COALESCE(pg_wal_lsn_diff(write_lsn,flush_lsn),0),COALESCE(pg_wal_lsn_diff(flush_lsn,replay_lsn),0),COALESCE(EXTRACT(EPOCH FROM now()-reply_time),0) FROM pg_stat_replication ORDER BY application_name,client_addr`
}
func replicationSlotsSQL(v int) string {
	inactive := "0"
	if v >= 170000 {
		inactive = `COALESCE(EXTRACT(EPOCH FROM now()-inactive_since),0)`
	}
	return `SELECT slot_name,slot_type,COALESCE(database,''),active,COALESCE(wal_status,''),COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(),restart_lsn),0),` + inactive + ` FROM pg_replication_slots ORDER BY slot_name`
}

func (c *Core) CollectWAL(ctx context.Context) (models.WALStats, error) {
	var version int
	var recovery bool
	if err := c.db.QueryRow(ctx, `SELECT current_setting('server_version_num')::int,pg_is_in_recovery()`).Scan(&version, &recovery); err != nil {
		return models.WALStats{}, err
	}
	out := models.WALStats{CollectedAt: time.Now().UTC()}
	if err := c.db.QueryRow(ctx, checkpointStatsSQL(version)).Scan(&out.TimedCheckpoints, &out.RequestedCheckpoints, &out.WriteTimeMS, &out.SyncTimeMS, &out.BuffersWritten, &out.StatsReset, &out.RestartpointsTimed, &out.RestartpointsRequested, &out.RestartpointsDone); err != nil {
		return out, err
	}
	if err := c.db.QueryRow(ctx, `SELECT wal_records,wal_fpi,wal_bytes,wal_buffers_full,stats_reset FROM pg_stat_wal`).Scan(&out.WALRecords, &out.WALFullPageImages, &out.WALBytes, &out.WALBuffersFull, &out.WALStatsReset); err != nil {
		return out, fmt.Errorf("WAL activity: %w", err)
	}
	if err := c.db.QueryRow(ctx, `SELECT current_setting('archive_mode'),current_setting('archive_command')<>'' OR current_setting('archive_library')<>'',archived_count,failed_count,COALESCE(last_archived_wal,''),last_archived_time,COALESCE(last_failed_wal,''),last_failed_time FROM pg_stat_archiver`).Scan(&out.ArchiveMode, &out.ArchiveConfigured, &out.ArchivedCount, &out.FailedArchiveCount, &out.LastArchivedWAL, &out.LastArchivedAt, &out.LastFailedWAL, &out.LastFailedAt); err != nil {
		return out, fmt.Errorf("archive activity: %w", err)
	}
	q := `SELECT pg_current_wal_lsn()::text`
	if recovery {
		q = `SELECT COALESCE(pg_last_wal_replay_lsn()::text,'')`
	}
	if err := c.db.QueryRow(ctx, q).Scan(&out.CurrentLSN); err != nil {
		return out, fmt.Errorf("WAL position: %w", err)
	}
	return out, nil
}
func checkpointStatsSQL(v int) string {
	if v >= 170000 {
		return `SELECT num_timed,num_requested,write_time,sync_time,buffers_written,stats_reset,restartpoints_timed,restartpoints_req,restartpoints_done FROM pg_stat_checkpointer`
	}
	return `SELECT checkpoints_timed,checkpoints_req,checkpoint_write_time,checkpoint_sync_time,buffers_checkpoint,stats_reset,0,0,0 FROM pg_stat_bgwriter`
}
