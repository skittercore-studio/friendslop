import { useState } from "preact/hooks";
import type { PublicCharacter } from "../types";

interface Props {
  characters: PublicCharacter[] | undefined;
}

export function CharacterPool({ characters }: Props) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (!characters || characters.length === 0) {
    return (
      <div class="char-pool empty">
        <em>Character pool will appear here once the game starts.</em>
      </div>
    );
  }

  return (
    <div class="char-pool">
      <h3 class="char-pool-title">Character pool</h3>
      <ul class="char-pool-list">
        {characters.map((c) => {
          const isExpanded = expandedId === c.id;
          const longBlurb = c.blurb.length > 80;
          return (
            <li
              key={c.id}
              class={isExpanded ? "char expanded" : "char"}
              onClick={() =>
                longBlurb ? setExpandedId(isExpanded ? null : c.id) : undefined
              }
              title={longBlurb ? "Click to expand" : undefined}
            >
              <strong class="char-name">{c.name}</strong>
              <span class="char-blurb">
                {!longBlurb || isExpanded
                  ? c.blurb
                  : c.blurb.slice(0, 78) + "…"}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
