CREATE TABLE tag (
    id INT AUTO_INCREMENT PRIMARY KEY,  -- Auto-incremented primary key
    tag VARCHAR(255) NOT NULL UNIQUE,   -- Unique tag field, cannot be null
    merchantID VARCHAR(255) NOT NULL,   -- Merchant ID field, cannot be null
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  -- Timestamp for record creation
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  -- Timestamp for record updates
);