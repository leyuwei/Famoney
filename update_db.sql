ALTER TABLE wallet_owners ADD COLUMN display_order INT DEFAULT 0;
UPDATE wallet_owners SET display_order=wallet_id;
