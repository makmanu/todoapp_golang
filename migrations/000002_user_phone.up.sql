BEGIN;
ALTER TABLE todoapp.users
    ADD COLUMN phone_number VARCHAR(15),
    ADD CONSTRAINT users_phone_number_format
        CHECK (phone_number ~ '^\+[1-9][0-9]{8,13}$');
COMMIT;
