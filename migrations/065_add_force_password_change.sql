-- Add force_password_change flag to users so password resets can require change on next login
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS force_password_change BOOLEAN DEFAULT FALSE;

-- Ensure existing rows have a non-null value
UPDATE users SET force_password_change = FALSE WHERE force_password_change IS NULL;
