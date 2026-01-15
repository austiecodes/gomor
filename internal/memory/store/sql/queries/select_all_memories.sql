SELECT id, text, tags, source, created_at, confidence, stability_days, last_retrieved_at, provider, model_id, dim, embedding
FROM memories
ORDER BY created_at DESC;
