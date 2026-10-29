import { FormControl, InputLabel, MenuItem, Select, SelectChangeEvent } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { GroupPermission } from "../../../api/user.ts";
import { getUserNodes } from "../../../api/api.ts";
import { UserNode } from "../../../api/dashboard.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import { GroupBS } from "../../../session/utils.ts";

export interface TaskNodeSelectProps {
  value: number;
  onChange: (nodeId: number) => void;
}

/**
 * TaskNodeSelect lets users with the select_node group permission pin a
 * background task to a specific processing node. Renders nothing otherwise.
 */
const TaskNodeSelect = ({ value, onChange }: TaskNodeSelectProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [nodes, setNodes] = useState<UserNode[]>([]);

  const canSelect = GroupBS(SessionManager.currentLoginOrNull()?.user).enabled(GroupPermission.select_node);

  useEffect(() => {
    if (!canSelect) return;
    dispatch(getUserNodes())
      .then((res) => setNodes(res.nodes ?? []))
      .catch(() => setNodes([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canSelect]);

  if (!canSelect) {
    return null;
  }

  return (
    <FormControl fullWidth margin="dense">
      <InputLabel>{t("fileManager.taskNode")}</InputLabel>
      <Select
        value={value}
        label={t("fileManager.taskNode")}
        onChange={(e: SelectChangeEvent<unknown>) => onChange(e.target.value as number)}
      >
        <MenuItem value={0}>{t("fileManager.taskNodeAuto")}</MenuItem>
        {nodes.map((n) => (
          <MenuItem key={n.id} value={n.id}>
            {n.name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};

export default TaskNodeSelect;
