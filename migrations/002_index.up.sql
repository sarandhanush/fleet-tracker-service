CREATE INDEX IF NOT EXISTS idx_vehicle_plate_number ON vehicle(plate_number);
CREATE INDEX IF NOT EXISTS idx_trips_vehicle_id ON trips(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_trips_time ON trips(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_vehicle_last_status_gin ON vehicle USING GIN (last_status);