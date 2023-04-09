-- name: FindMemberByID :one
SELECT * FROM members
WHERE id = ? LIMIT 1;

-- name: FindMember :one
SELECT * FROM members
WHERE slack_uid = ? AND month = ? AND year = ? LIMIT 1;

-- name: MostLikesReceived :many
SELECT m.* 
FROM members m
WHERE m.month = ? 
  AND m.year = ? 
  AND m.received_likes = (
      SELECT MAX(m2.received_likes) 
      FROM members m2
      WHERE m2.month = ? 
        AND m2.year = ?
  );

-- name: MostDislikesReceived :many
SELECT m.*
FROM members m
WHERE m.month = ?
  AND m.year = ?
  AND m.received_dislikes = (
      SELECT MAX(m2.received_dislikes)
      FROM members m2
      WHERE m2.month = ?
        AND m2.year = ?
  );

-- name: MostReactionsReceived :many
SELECT m.*
FROM members m
WHERE m.month = ?
  AND m.year = ?
  AND m.received_reactions = (
      SELECT MAX(m2.received_reactions)
      FROM members m2
      WHERE m2.month = ?
        AND m2.year = ?
  );

-- name: MostLikesGiven :many
SELECT m.*
FROM members m
WHERE m.month = ?
  AND m.year = ?
  AND m.given_likes = (
      SELECT MAX(m2.given_likes)
      FROM members m2
      WHERE m2.month = ?
        AND m2.year = ?
  );

-- name: MostDislikesGiven :many
SELECT m.*
FROM members m
WHERE m.month = ?
  AND m.year = ?
  AND m.given_dislikes = (
      SELECT MAX(m2.given_dislikes)
      FROM members m2
      WHERE m2.month = ?
        AND m2.year = ?
  );

-- name: MostReactionsGiven :many
SELECT m.*
FROM members m
WHERE m.month = ?
  AND m.year = ?
  AND m.given_reactions = (
      SELECT MAX(m2.given_reactions)
      FROM members m2
      WHERE m2.month = ?
        AND m2.year = ?
  );

-- name: CreateMember :one
INSERT INTO members (
    slack_uid,
    month,
    year
) VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: UpdateReceivedLikes :exec
UPDATE members
SET received_likes = ?
WHERE id = ?;

-- name: UpdateReceivedDislikes :exec
UPDATE members
SET received_dislikes = ?
WHERE id = ?;

-- name: UpdateReceivedReactions :exec
UPDATE members
SET received_reactions = ?
WHERE id = ?;

-- name: UpdateGivenLikes :exec
UPDATE members
SET given_likes = ?
WHERE id = ?;

-- name: UpdateGivenDislikes :exec
UPDATE members
SET given_dislikes = ?
WHERE id = ?;

-- name: UpdateGivenReactions :exec
UPDATE members
SET given_reactions = ?
WHERE id = ?;

-- name: UpdateMember :exec
UPDATE members
SET received_likes = ?,
received_dislikes = ?,
received_reactions = ?,
given_likes = ?,
given_dislikes = ?,
given_reactions = ?
WHERE id = ?;


-- name: DeleteMember :exec
DELETE FROM members
WHERE id = ?;
