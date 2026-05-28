ALTER TABLE evidence_reports ADD COLUMN smoke_summaries_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE evidence_reports ADD COLUMN smoke_matrix_json TEXT NOT NULL DEFAULT '[]';
