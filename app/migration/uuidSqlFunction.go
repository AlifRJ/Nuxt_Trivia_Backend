package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func UuidSQLFunction(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	create or replace function uuid_generate_v7()
	returns uuid
	language plpgsql
	as $$
	declare
		v_time timestamp with time zone := clock_timestamp();
		v_bytes bytea;
		v_epoch_msec bigint;
	begin
		-- Calculate 48-bit Unix epoch time in milliseconds
		v_epoch_msec := (extract(epoch from v_time) * 1000)::bigint;

		-- Generate 16 random bytes
		v_bytes := uuid_send(gen_random_uuid());

		-- Overlay the first 6 bytes with the 48-bit millisecond timestamp
		v_bytes := set_byte(v_bytes, 0, (v_epoch_msec >> 40 & 255)::int);
		v_bytes := set_byte(v_bytes, 1, (v_epoch_msec >> 32 & 255)::int);
		v_bytes := set_byte(v_bytes, 2, (v_epoch_msec >> 24 & 255)::int);
		v_bytes := set_byte(v_bytes, 3, (v_epoch_msec >> 16 & 255)::int);
		v_bytes := set_byte(v_bytes, 4, (v_epoch_msec >> 8 & 255)::int);
		v_bytes := set_byte(v_bytes, 5, (v_epoch_msec & 255)::int);

		-- Set Version to 7 (0111xxxx in byte 6)
		v_bytes := set_byte(v_bytes, 6, (get_byte(v_bytes, 6) & 15) | 112);

		-- Set Variant to RFC 9562 / DCE 118 (10xxxxxx in byte 8)
		v_bytes := set_byte(v_bytes, 8, (get_byte(v_bytes, 8) & 63) | 128);

		-- Encode to UUID
		return encode(v_bytes, 'hex')::uuid;
	end;
	$$;
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run custom uuid7 function for postgresql: %w", err)
	}
	return nil
}