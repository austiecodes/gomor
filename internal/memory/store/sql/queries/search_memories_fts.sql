SELECT m.id, m.text, m.tags, m.source, m.created_at,
       m.confidence, m.stability_days, m.last_retrieved_at,
       m.provider, m.model_id, m.dim, m.embedding,
       snippet(memories_fts, 0, '>>>', '<<<', '...', 32) as snippet,
       rank
FROM memories m
JOIN memories_fts fts ON m.rowid = fts.rowid
WHERE memories_fts MATCH ?
ORDER BY rank
LIMIT ?;
