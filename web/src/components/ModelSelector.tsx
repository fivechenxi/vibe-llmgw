interface Props {
  models: { model_id: string; remaining_tokens: number }[];
  selected: string;
  onChange: (modelId: string) => void;
}

export function ModelSelector({ models, selected, onChange }: Props) {
  return (
    <select
      value={selected}
      onChange={(e) => onChange(e.target.value)}
      className="border rounded px-2 py-1 text-sm"
    >
      {models.map((m) => (
        <option key={m.model_id} value={m.model_id}>
          {m.model_id} ({m.remaining_tokens.toLocaleString()} tokens left)
        </option>
      ))}
    </select>
  );
}
