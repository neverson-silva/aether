-- Compose stacks podem pertencer a um Environment, permitindo que recebam
-- Environment Variables (além das Project Variables) via resolução do resolver.
ALTER TABLE compose_apps ADD COLUMN environment_id UUID REFERENCES environments(id) ON DELETE SET NULL;
