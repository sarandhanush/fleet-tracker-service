
# Query Performance with and without Index

This document shows the difference in query execution when running:

```sql
EXPLAIN ANALYZE
SELECT id FROM trips 
WHERE vehicle_id = '550e8400-e29b-41d4-a716-446655440002' 
  AND start_time >= now() - interval '24 hours';
```

## Without Index

-- ref ./exp_analyze_without_index.png

- **Execution Plan**: Sequential Scan (Seq Scan)
- **Execution Time**: ~7.936 ms

---

## With Index

-- ref ./exp_analyze_with_index.png

- **Execution Plan**: Bitmap Index Scan
- **Execution Time**: ~5.446 ms

--
Adding an index on `(vehicle_id, start_time)` improves performance by reducing execution time and avoiding full table scans.
