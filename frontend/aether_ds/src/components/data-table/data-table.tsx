import { CaretUpDown, Check } from "@phosphor-icons/react";
import { type ReactNode, useMemo, useState } from "react";
export interface DataTableColumn<T> {
  id: string;
  header: ReactNode;
  accessor: (row: T) => ReactNode;
  sortValue?: (row: T) => string | number;
  width?: string;
  editable?: boolean;
  pinned?: "left" | "right";
}
export interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  data: T[];
  rowId: (row: T) => string;
  selectable?: boolean;
  selectedIds?: string[];
  onSelectionChange?: (ids: string[]) => void;
  empty?: ReactNode;
  loading?: boolean;
  page?: number;
  pageSize?: number;
  onPageChange?: (page: number) => void;
  onCellEdit?: (row: T, columnId: string, value: string) => void;
  resizable?: boolean;
}
export function DataTable<T>({
  columns,
  data,
  empty = "No records found.",
  loading,
  onSelectionChange,
  rowId,
  selectable,
  selectedIds = [],
  onCellEdit,
  onPageChange,
  page = 1,
  pageSize,
  resizable = false,
}: DataTableProps<T>) {
  const [sort, setSort] = useState<{
    id: string;
    direction: "asc" | "desc";
  } | null>(null);
  const rows = useMemo(() => {
    if (!sort) return data;
    const column = columns.find((item) => item.id === sort.id);
    if (!column?.sortValue) return data;
    return [...data].sort((a, b) => {
      const left = column.sortValue?.(a) ?? "";
      const right = column.sortValue?.(b) ?? "";
      return (
        (left < right ? -1 : left > right ? 1 : 0) *
        (sort.direction === "asc" ? 1 : -1)
      );
    });
  }, [columns, data, sort]);
  const pagedRows = pageSize
    ? rows.slice((page - 1) * pageSize, page * pageSize)
    : rows;
  const pageCount = pageSize
    ? Math.max(1, Math.ceil(rows.length / pageSize))
    : 1;
  const allSelected =
    pagedRows.length > 0 &&
    pagedRows.every((row) => selectedIds.includes(rowId(row)));
  const toggleAll = () =>
    onSelectionChange?.(allSelected ? [] : pagedRows.map(rowId));
  const toggleRow = (id: string) =>
    onSelectionChange?.(
      selectedIds.includes(id)
        ? selectedIds.filter((item) => item !== id)
        : [...selectedIds, id],
    );
  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-surface-card shadow-sm">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[42rem] border-collapse text-body-sm">
          <thead className="bg-surface-container-low/80 text-label-caps text-muted-foreground">
            <tr>
              {selectable ? (
                <th scope="col" className="w-12 px-4 py-3">
                  <button
                    type="button"
                    aria-label="Select all rows"
                    aria-pressed={allSelected}
                    onClick={toggleAll}
                    className="inline-flex size-4 items-center justify-center rounded border border-border text-primary outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary focus-visible:ring-2 focus-visible:ring-ring active:scale-95 data-[selected=true]:border-primary data-[selected=true]:bg-primary"
                    data-selected={allSelected}
                  >
                    {allSelected ? <Check size={12} /> : null}
                  </button>
                </th>
              ) : null}
              {columns.map((column) => (
                <th
                  key={column.id}
                  scope="col"
                  className={`relative px-4 py-3 text-start ${resizable ? "resize-x overflow-hidden" : ""} ${column.pinned === "left" ? "sticky left-0 z-10 bg-surface-container-low" : column.pinned === "right" ? "sticky right-0 z-10 bg-surface-container-low" : ""}`}
                  style={{ width: column.width }}
                >
                  {column.sortValue ? (
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 rounded-lg px-1.5 py-1 font-semibold outline-none transition-[background-color,color] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() =>
                        setSort((current) =>
                          current?.id === column.id &&
                          current.direction === "asc"
                            ? { id: column.id, direction: "desc" }
                            : { id: column.id, direction: "asc" },
                        )
                      }
                    >
                      {column.header}
                      <CaretUpDown size={14} />
                    </button>
                  ) : (
                    column.header
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {loading ? (
              <tr>
                <td
                  colSpan={columns.length + (selectable ? 1 : 0)}
                  className="px-4 py-8 text-center text-muted-foreground"
                >
                  Loading records...
                </td>
              </tr>
            ) : pagedRows.length ? (
              pagedRows.map((row) => {
                const id = rowId(row);
                const selected = selectedIds.includes(id);
                return (
                  <tr
                    key={id}
                    data-selected={selected}
                    className="transition-[background-color,box-shadow] duration-200 hover:bg-surface-container data-[selected=true]:bg-primary/10"
                  >
                    {selectable ? (
                      <td className="px-4 py-3">
                        <button
                          type="button"
                          aria-label={`Select row ${id}`}
                          aria-pressed={selected}
                          onClick={() => toggleRow(id)}
                          className="inline-flex size-4 items-center justify-center rounded border border-border text-primary outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary focus-visible:ring-2 focus-visible:ring-ring active:scale-95 data-[selected=true]:border-primary data-[selected=true]:bg-primary data-[selected=true]:text-primary-foreground"
                          data-selected={selected}
                        >
                          {selected ? <Check size={12} /> : null}
                        </button>
                      </td>
                    ) : null}
                    {columns.map((column) => (
                      <td
                        key={column.id}
                        className={`px-4 py-3 text-foreground ${column.pinned === "left" ? "sticky left-0 z-[1] bg-surface-card" : column.pinned === "right" ? "sticky right-0 z-[1] bg-surface-card" : ""}`}
                      >
                        {column.editable && onCellEdit ? (
                          <input
                            defaultValue={String(column.accessor(row) ?? "")}
                            onBlur={(event) =>
                              onCellEdit(row, column.id, event.target.value)
                            }
                            className="w-full min-w-24 rounded-lg border border-transparent bg-transparent px-2 py-1 outline-none transition-[background-color,border-color,box-shadow] duration-150 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-primary/20"
                          />
                        ) : (
                          column.accessor(row)
                        )}
                      </td>
                    ))}
                  </tr>
                );
              })
            ) : (
              <tr>
                <td
                  colSpan={columns.length + (selectable ? 1 : 0)}
                  className="px-4 py-8 text-center text-muted-foreground"
                >
                  {empty}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {pageSize && pageCount > 1 ? (
        <div className="flex items-center justify-between border-t border-border bg-surface-container-lowest/50 px-4 py-3 text-body-sm text-muted-foreground">
          <span>
            Page {page} of {pageCount}
          </span>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={page <= 1}
              onClick={() => onPageChange?.(page - 1)}
              className="rounded-xl border border-border px-3 py-1.5 outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:opacity-40"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={page >= pageCount}
              onClick={() => onPageChange?.(page + 1)}
              className="rounded-xl border border-border px-3 py-1.5 outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
