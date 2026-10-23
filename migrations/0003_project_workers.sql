-- 0003_project_workers.sql - replace per-day attendance project tags with
-- project-level worker assignments.

CREATE TABLE project_workers (
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worker_id  INTEGER NOT NULL REFERENCES workers(id),
    PRIMARY KEY (project_id, worker_id)
) STRICT;

CREATE INDEX idx_project_workers_worker ON project_workers (worker_id);

-- Preserve the project and worker pairs already recorded per attendance day.
INSERT OR IGNORE INTO project_workers (project_id, worker_id)
SELECT DISTINCT ap.project_id, a.worker_id
FROM attendance_projects ap
JOIN attendances a ON a.id = ap.attendance_id;

DROP TABLE attendance_projects;
