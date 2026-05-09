# content/

Static seed content for friendslop. Loaded once at server start (or baked into the binary via `embed.FS`); not mutable from the running game in MVP.

Two banks here, both JSON arrays:

- `characters_default.json` — the default character pool used when a room is created with `pool_source: curated`.
- `questions_default.json` — the default question bank used for round questions.

If a room is created with `pool_source: playerwritten`, characters are submitted at runtime instead and this file is unused for that game. Questions are always drawn from `questions_default.json` in MVP.

## Format

### `characters_default.json`

Array of objects. One entry per character.

```json
[
  {
    "template_id": "char_001",
    "name": "Sherlock Holmes",
    "blurb": "Brilliant Victorian detective; cocaine habit; speaks down to everyone."
  }
]
```

| Field         | Type   | Required | Notes                                                                                          |
|---------------|--------|----------|------------------------------------------------------------------------------------------------|
| `template_id` | string | yes      | Stable identifier. Format `char_NNN` (zero-padded). Must be unique within the file.            |
| `name`        | string | yes      | Display name. 1-60 chars. Shown verbatim in the public pool.                                   |
| `blurb`       | string | yes      | One-line voice-and-vibe primer. 60-200 chars. No newlines. Shown publicly under the name.      |

The blurb is a primer, not a dossier — it's what reminds players who the character is and how they speak. Voice should leak out of the blurb itself ("Brilliant Victorian detective; cocaine habit; speaks down to everyone." > "A famous detective.").

### `questions_default.json`

Array of objects. One entry per question.

```json
[
  { "id": "q_001", "text": "Describe your worst Monday morning." }
]
```

| Field  | Type   | Required | Notes                                                                                            |
|--------|--------|----------|--------------------------------------------------------------------------------------------------|
| `id`   | string | yes      | Stable identifier. Format `q_NNN` (zero-padded). Must be unique within the file.                 |
| `text` | string | yes      | The prompt the player sees. Open-ended, voice-revealing, story-prompting.                        |

Bias toward questions that work cross-character (a finance bro, Gandalf, and a doomer can all answer "describe your morning routine"). Avoid prompts that only work for one franchise or genre, factual prompts, yes/no questions, and one-word-answer questions.

## Adding entries

1. Open the relevant file.
2. Append a new object to the array. Pick the next free `template_id` / `id`. Zero-pad to three digits.
3. Validate locally:

   ```sh
   jq . content/characters_default.json > /dev/null
   jq . content/questions_default.json > /dev/null
   jq '[.[].template_id] | unique | length == length' content/characters_default.json
   jq '[.[].id]          | unique | length == length' content/questions_default.json
   ```

4. Commit. The server reads these at startup; restart to pick up changes.

## Quality bar

**Characters**

- Mix categories: well-known fiction, archetypes, internet/meme types, distinctive-voice formats (announcer, narrator, etc.).
- Don't over-index on one franchise.
- No copyrighted scripts or extended lore — these are parody-tier brief descriptions, kept short and recognisable.
- No slurs, no explicit content, no targeted-at-real-people stuff.
- Funny is good. Specific is better than generic. Voice should be inferable from the blurb alone.

**Questions**

- Open-ended and evocative. Stories beat answers.
- Cross-character: ask things a wide range of personalities can credibly respond to.
- No factual / trivia prompts ("What is the capital of France?").
- No yes/no prompts ("Do you like cheese?").
- No prompts that demand a one-word answer.

## IDs and stability

`template_id` and `id` are referenced from `room_characters.template_id` and `rounds.question_text` respectively (the latter snapshots the text at game start, so question rewrites are safe; the former is a foreign key into this file, so don't reuse or recycle ids — only append).

If you need to retire an entry, leave the id reserved and remove the row, or replace the content under the same id if it's a clear edit (typo fix, grammar). Don't shuffle ids.
