interface Column<T> {
  key: string;
  label: string;
  render: (row: T) => React.ReactNode;
  className?: string;
}

interface TableProps<T> {
  columns: Column<T>[];
  data: T[];
  keyFn: (row: T) => string;
  onRowClick?: (row: T) => void;
  emptyMessage?: string;
}

export function Table<T>({ columns, data, keyFn, onRowClick, emptyMessage = 'No data' }: TableProps<T>) {
  if (!data || data.length === 0) {
    return <p className="p-4 text-bc-muted text-sm">{emptyMessage}</p>;
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-bc-border text-left">
          {columns.map((col) => (
            <th key={col.key} className={`px-4 py-2 font-medium text-bc-muted ${col.className ?? ''}`}>
              {col.label}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {data.map((row) => (
          <tr
            key={keyFn(row)}
            onClick={onRowClick ? () => onRowClick(row) : undefined}
            className={`border-b border-bc-border/50 ${
              onRowClick ? 'cursor-pointer hover:bg-bc-surface' : ''
            }`}
          >
            {columns.map((col) => (
              <td key={col.key} className={`px-4 py-2 ${col.className ?? ''}`}>
                {col.render(row)}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
