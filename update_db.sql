ALTER TABLE flows ADD COLUMN operator_id INT NULL AFTER description;
ALTER TABLE flows ADD CONSTRAINT fk_flows_operator FOREIGN KEY (operator_id) REFERENCES users(id);
