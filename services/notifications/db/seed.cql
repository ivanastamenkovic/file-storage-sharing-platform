INSERT INTO notifications.notifications (user_id, created_at, id, type, message, is_read)
VALUES ('user1', '2026-11-16 09:00:00+0000', 11111111-1111-1111-1111-111111111111, 'SHARED', 'user2 shared the file "report.pdf" with you', true);

INSERT INTO notifications.notifications (user_id, created_at, id, type, message, is_read)
VALUES ('user1', '2026-11-16 10:30:00+0000', 11111111-1111-1111-1111-111111111112, 'NEW_FILE_IN_FOLDER', 'A new file "notes.txt" was added to the folder "Project"', false);

INSERT INTO notifications.notifications (user_id, created_at, id, type, message, is_read)
VALUES ('user1', '2026-11-16 12:15:00+0000', 11111111-1111-1111-1111-111111111113, 'NEW_VERSION', 'A new version of "report.pdf" was uploaded', false);

INSERT INTO notifications.notifications (user_id, created_at, id, type, message, is_read)
VALUES ('user2', '2026-11-16 08:45:00+0000', 22222222-2222-2222-2222-222222222221, 'SHARED', 'user1 shared the folder "Photos" with you', false);

INSERT INTO notifications.notifications (user_id, created_at, id, type, message, is_read)
VALUES ('user2', '2026-11-16 11:00:00+0000', 22222222-2222-2222-2222-222222222222, 'NEW_VERSION', 'A new version of "budget.xlsx" was uploaded', false);
