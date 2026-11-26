BEGIN;
ALTER TABLE todoapp.users DROP CONSTRAINT users_phone_number_format, DROP COLUMN phone_number;
COMMIT;
