-- Attendance fractions are configurable: 1 = quarter, 2 = half, 3 = three
-- quarters. Full days and absences are always available.

INSERT INTO settings (key, value)
VALUES ('attendance_fractions', '1,2,3')
ON CONFLICT (key) DO NOTHING;
