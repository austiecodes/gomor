package store

import _ "embed"

// Schema SQL
//
//go:embed sql/schema/schema.sql
var schemaSQL string

// Query SQL - each file contains a single query
var (
	//go:embed sql/queries/insert_memory.sql
	insertMemorySQL string
	//go:embed sql/queries/select_all_memories.sql
	selectAllMemoriesSQL string
	//go:embed sql/queries/delete_memory.sql
	deleteMemorySQL string
	//go:embed sql/queries/update_memory_embedding.sql
	updateMemoryEmbeddingSQL string
	//go:embed sql/queries/update_memory_decay.sql
	updateMemoryDecaySQL string
	//go:embed sql/queries/search_memories_fts.sql
	searchMemoriesFTSSQL string
	//go:embed sql/queries/clear_memories.sql
	clearMemoriesSQL string
	//go:embed sql/queries/insert_history.sql
	insertHistorySQL string
	//go:embed sql/queries/search_history_fts.sql
	searchHistoryFTSSQL string
	//go:embed sql/queries/select_recent_history.sql
	selectRecentHistorySQL string
	//go:embed sql/queries/clear_history.sql
	clearHistorySQL string
)
