# Word Selection Guide

Use this guide when adding or reviewing words for Impostor Game.

## Goal

Good words create fun table talk: players can give clues that are related, but not so obvious that the impostor is instantly lost. Words should be easy to understand, PG, kid-friendly, and fun to say out loud.

## Good Word Criteria

Choose words that are:

- **PG and kid-friendly**: safe for family play, classrooms, and mixed ages.
- **Concrete and familiar**: things most players can picture quickly.
- **Fun to hint around**: words with many possible related clues.
- **Not too obscure**: avoid words that only some players will know.
- **Not too broad**: avoid vague words like `thing`, `place`, `fun`, or `game`.
- **Not too narrow**: avoid overly specific proper nouns, brands, or niche terms.
- **Easy to pronounce**: players should be able to say and understand them aloud.
- **One clear idea**: short phrases are okay when natural, like `ice cream` or `water park`.

## Category Fit

Put each word in the category where it fits best. A word should appear in only one category.

Examples:

- `teacher` belongs in **School**, not Jobs, because it is strongly tied to the school setting.
- `waterfall` belongs in **Nature**, not Places.
- `backpack` belongs in **School**, not Objects.
- `swimming` belongs in **Sports**, not Activities.

## Duplicate Policy

Avoid duplicate words across all categories, not just within one category.

When a duplicate exists:

1. Keep it in the category where it creates the strongest clue space.
2. Replace it in the weaker category with a new, high-quality word.
3. Prefer replacing duplicates over deleting them, unless the category already has plenty of strong words.

## What to Avoid

Avoid words that are:

- Scary, violent, adult, political, religious, or controversial.
- Gross or mean-spirited.
- Brand names, celebrities, franchises, or copyrighted characters.
- Too abstract, like `justice`, `energy`, or `freedom`.
- Too similar to existing words in the same category.
- Likely to create one obvious clue only.

## Quality Check Before Adding

Before committing new words:

1. Read the category aloud and remove weak words.
2. Check for duplicates across all categories.
3. Make sure every word is PG and kid-friendly.
4. Make sure every word gives players several possible clue directions.
5. Run tests.

## Helpful Command

Use this script to check duplicate words across categories in `main.go`:

```bash
python3 - <<'PY'
import ast, collections, re
s = open('main.go').read()
block = re.search(r'var words = map\[string\]\[\]string\{(.*?)\n\}', s, re.S).group(1)
seen = collections.defaultdict(list)
for category, items in re.findall(r'"([^"]+)":\s*\{([^}]*)\}', block):
    for word in ast.literal_eval('[' + items + ']'):
        seen[word].append(category)
for word, categories in sorted(seen.items()):
    if len(categories) > 1:
        print(f'{word}: {categories}')
PY
```

No output means there are no duplicate words across categories.
