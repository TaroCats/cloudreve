import {
  Box,
  FormControl,
  FormControlLabel,
  Radio,
  RadioGroup,
  SelectChangeEvent,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getStoragePolicyList } from "../../../../api/api";
import { StoragePolicy } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import FacebookCircularProgress from "../../../Common/CircularProgress";
import SizeInput from "../../../Common/SizeInput";
import { DenseSelect, SquareChip } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";

export interface PolicySelectionInputProps {
  /** IDs of the policies bound to this group. */
  value: number[];
  /** ID of the policy used when nothing else resolves one. */
  defaultPolicyId: number;
  /** Per-policy usage caps in bytes; 0 / missing = only the group quota applies. */
  quotas: Record<number, number>;
  onChange: (value: number[], defaultPolicyId: number, quotas: Record<number, number>) => void;
}

/**
 * PolicySelectionInput lets an admin bind several storage policies to a group,
 * mark one as the default, and optionally cap usage per policy.
 */
const PolicySelectionInput = ({ value, defaultPolicyId, quotas, onChange }: PolicySelectionInputProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);
  const [loading, setLoading] = useState(false);

  const policyMap = useMemo(
    () =>
      policies.reduce(
        (acc, policy) => {
          acc[policy.id] = policy;
          return acc;
        },
        {} as Record<number, StoragePolicy>,
      ),
    [policies],
  );

  const handleChange = useCallback(
    (event: SelectChangeEvent<unknown>) => {
      const selected = (event.target.value as number[]).filter((id) => id > 0);
      // Keep the default inside the selection; otherwise fall back to the first.
      const nextDefault = selected.includes(defaultPolicyId) ? defaultPolicyId : (selected[0] ?? 0);
      // Drop quota entries for policies that are no longer bound.
      const nextQuotas = Object.fromEntries(Object.entries(quotas).filter(([id]) => selected.includes(Number(id))));
      onChange(selected, nextDefault, nextQuotas);
    },
    [defaultPolicyId, quotas, onChange],
  );

  const onDefaultChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange(value, Number(e.target.value), quotas);
    },
    [value, quotas, onChange],
  );

  const onQuotaChange = useCallback(
    (policyId: number, size: number) => {
      onChange(value, defaultPolicyId, { ...quotas, [policyId]: size });
    },
    [value, defaultPolicyId, quotas, onChange],
  );

  useEffect(() => {
    setLoading(true);
    dispatch(getStoragePolicyList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" }))
      .then((res) => setPolicies(res.policies))
      .finally(() => setLoading(false));
  }, [dispatch]);

  return (
    <Box>
      <FormControl fullWidth>
        <DenseSelect
          multiple
          value={value}
          required
          onChange={handleChange}
          sx={{ minHeight: 39 }}
          disabled={loading}
          MenuProps={{
            PaperProps: { sx: { maxWidth: 230 } },
            MenuListProps: {
              sx: {
                "& .MuiMenuItem-root": { whiteSpace: "normal" },
              },
            },
          }}
          renderValue={(value) => {
            const selected = (value as number[]) ?? [];
            return (
              <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                {loading ? (
                  <FacebookCircularProgress size={20} sx={{ mt: "1px" }} />
                ) : (
                  selected.map((id) => <SquareChip size="small" key={id} label={policyMap[id]?.name ?? id} />)
                )}
              </Box>
            );
          }}
        >
          {policies.map((policy) => (
            <SquareMenuItem key={policy.id} value={policy.id}>
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                <Typography variant={"body2"} fontWeight={600}>
                  {policy.name}
                </Typography>
                <Typography variant={"caption"} color={"textSecondary"}>
                  {t(`policy.${policy.type}`)}
                </Typography>
              </Box>
            </SquareMenuItem>
          ))}
        </DenseSelect>
      </FormControl>

      {value.length > 0 && (
        <Box sx={{ mt: 1.5, borderLeft: "2px solid", borderColor: "divider", pl: 1.5 }}>
          <Typography variant="caption" color="textSecondary">
            {t("group.defaultPolicyDes")}
          </Typography>
          <RadioGroup value={String(defaultPolicyId)} onChange={onDefaultChange}>
            {value.map((id) => (
              <Box key={id} sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 2 }}>
                <FormControlLabel
                  value={String(id)}
                  control={<Radio size="small" />}
                  label={policyMap[id]?.name ?? id}
                  sx={{ flex: 1, mr: 0 }}
                />
                <Box sx={{ width: 200, pb: 0.5 }}>
                  <SizeInput
                    variant="outlined"
                    value={quotas[id] ?? 0}
                    onChange={(size) => onQuotaChange(id, size)}
                    label={t("group.policyQuota")}
                    allowZero
                  />
                </Box>
              </Box>
            ))}
          </RadioGroup>
          <Typography variant="caption" color="textSecondary">
            {t("group.policyQuotaDes")}
          </Typography>
        </Box>
      )}
    </Box>
  );
};

export default PolicySelectionInput;
