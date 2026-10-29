import { Checkbox, ListItemText } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getNodeList } from "../../../../api/api";
import { Node } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import { DenseSelect } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";

export interface MultipleNodeSelectionInputProps {
  value: number[];
  onChange: (nodeIds: number[]) => void;
}

const MultipleNodeSelectionInput = ({ value, onChange }: MultipleNodeSelectionInputProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [nodes, setNodes] = useState<Node[]>([]);

  useEffect(() => {
    dispatch(getNodeList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" }))
      .then((res) => setNodes(res.nodes ?? []))
      .catch(() => setNodes([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <DenseSelect
      multiple
      displayEmpty
      value={value.map((id) => id.toString())}
      onChange={(e) => {
        const selected = typeof e.target.value === "string" ? e.target.value.split(",") : (e.target.value as string[]);
        onChange(selected.filter((v: string) => v !== "").map((v: string) => parseInt(v)));
      }}
      sx={{
        minHeight: 39,
      }}
      MenuProps={{
        PaperProps: { sx: { maxWidth: 230 } },
        MenuListProps: {
          sx: {
            "& .MuiMenuItem-root": {
              whiteSpace: "normal",
            },
          },
        },
      }}
      renderValue={(selected: unknown) => {
        const selectedIds = (selected as string[]) ?? [];
        if (selectedIds.length === 0) {
          return (
            <ListItemText
              primary={<em>{t("group.allNodes")}</em>}
              slotProps={{
                primary: { color: "textSecondary", variant: "body2" },
              }}
            />
          );
        }
        const names = nodes.filter((n) => selectedIds.includes(n.id?.toString())).map((n) => n.name);
        return (
          <ListItemText
            primary={names.join(", ")}
            slotProps={{
              primary: { variant: "body2" },
            }}
          />
        );
      }}
    >
      {nodes.map((node) => (
        <SquareMenuItem key={node.id} value={node.id?.toString()}>
          <Checkbox size={"small"} checked={value.includes(node.id)} />
          <ListItemText
            primary={node.name}
            secondary={`#${node.id} · ${node.type}`}
            slotProps={{
              primary: { variant: "body2" },
              secondary: { variant: "caption" },
            }}
          />
        </SquareMenuItem>
      ))}
    </DenseSelect>
  );
};

export default MultipleNodeSelectionInput;
