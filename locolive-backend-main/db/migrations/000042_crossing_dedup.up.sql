-- Prevent duplicate crossings for the same pair in the same 10-minute window
ALTER TABLE crossings ADD CONSTRAINT unique_crossing_pair_time UNIQUE (user_id_1, user_id_2, occurred_at);
