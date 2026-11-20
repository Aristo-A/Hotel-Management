-- Enable required PostgreSQL extensions as specified in BACKEND_INTEGRATION.md
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Set timezone to Asia/Kolkata (Indian Standard Time)
SET timezone = 'Asia/Kolkata';
