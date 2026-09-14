UPDATE users
SET global_role = 'admin'
WHERE id = (
    SELECT id
    FROM users
    ORDER BY created_at, id
    LIMIT 1
)
AND global_role <> 'admin';
