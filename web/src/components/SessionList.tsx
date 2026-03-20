interface Props {
  sessions: string[];
  current: string;
  onSelect: (id: string) => void;
  onNew: () => void;
}

export function SessionList({ sessions, current, onSelect, onNew }: Props) {
  return (
    <div className="w-56 border-r flex flex-col">
      <div className="p-3 border-b">
        <button onClick={onNew} className="w-full bg-blue-500 text-white rounded py-1 text-sm">
          + 新对话
        </button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {sessions.map((id) => (
          <div
            key={id}
            onClick={() => onSelect(id)}
            className={`px-3 py-2 text-sm cursor-pointer truncate hover:bg-gray-50 ${
              id === current ? "bg-blue-50 font-medium" : ""
            }`}
          >
            {id.slice(0, 8)}…
          </div>
        ))}
      </div>
    </div>
  );
}
