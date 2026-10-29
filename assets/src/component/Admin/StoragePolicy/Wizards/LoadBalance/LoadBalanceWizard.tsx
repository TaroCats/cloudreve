import { Button, FormControl, Typography } from "@mui/material";
import Box from "@mui/material/Box";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getStoragePolicyList } from "../../../../../api/api";
import { LoadBalanceStrategy, StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import FacebookCircularProgress from "../../../../Common/CircularProgress";
import { DenseFilledTextField, DenseSelect, SquareChip } from "../../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../../FileManager/ContextMenu/ContextMenu";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";

const LoadBalanceWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const formRef = useRef<HTMLFormElement>(null);
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    name: "",
    type: PolicyType.load_balance,
    settings: {
      policy_ids: [],
      strategy: LoadBalanceStrategy.round_robin,
    },
    edges: {},
  });

  useEffect(() => {
    setLoading(true);
    dispatch(getStoragePolicyList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" }))
      .then((res) =>
        // A load balance pool can only hold concrete storage policies.
        setPolicies(res.policies.filter((p) => p.type !== PolicyType.load_balance)),
      )
      .finally(() => setLoading(false));
  }, [dispatch]);

  const memberIds = policy.settings?.policy_ids ?? [];

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }
    onSubmit(policy);
  };

  return (
    <form ref={formRef} onSubmit={handleSubmit}>
      <SettingForm title={t("policy.name")} lgWidth={12}>
        <DenseFilledTextField
          fullWidth
          required
          value={policy.name}
          onChange={(e) => setPolicy({ ...policy, name: e.target.value })}
        />
        <NoMarginHelperText>{t("policy.policyName")}</NoMarginHelperText>
      </SettingForm>
      <SettingForm title={t("policy.memberPolicies")} lgWidth={12}>
        <FormControl fullWidth>
          <DenseSelect
            multiple
            required
            value={memberIds}
            disabled={loading}
            onChange={(e) =>
              setPolicy({
                ...policy,
                settings: {
                  ...policy.settings,
                  policy_ids: (e.target.value as number[]).filter((id) => id > 0),
                },
              })
            }
            sx={{ minHeight: 39 }}
            renderValue={(value) => {
              const selected = (value as number[]) ?? [];
              return (
                <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                  {loading ? (
                    <FacebookCircularProgress size={20} sx={{ mt: "1px" }} />
                  ) : (
                    selected.map((id) => (
                      <SquareChip
                        size="small"
                        key={id}
                        label={policies.find((p) => p.id === id)?.name ?? id}
                      />
                    ))
                  )}
                </Box>
              );
            }}
          >
            {policies.map((p) => (
              <SquareMenuItem key={p.id} value={p.id}>
                <Typography variant={"body2"} fontWeight={600}>
                  {p.name}
                </Typography>
                <Typography variant={"caption"} color={"textSecondary"}>
                  {t(`policy.${p.type}`)}
                </Typography>
              </SquareMenuItem>
            ))}
          </DenseSelect>
        </FormControl>
        <NoMarginHelperText>{t("policy.memberPoliciesDes")}</NoMarginHelperText>
      </SettingForm>
      <SettingForm title={t("policy.strategy")} lgWidth={12}>
        <FormControl fullWidth>
          <DenseSelect
            value={policy.settings?.strategy ?? LoadBalanceStrategy.round_robin}
            onChange={(e) =>
              setPolicy({
                ...policy,
                settings: {
                  ...policy.settings,
                  strategy: e.target.value as LoadBalanceStrategy,
                },
              })
            }
          >
            <SquareMenuItem value={LoadBalanceStrategy.round_robin}>{t("policy.roundRobin")}</SquareMenuItem>
            <SquareMenuItem value={LoadBalanceStrategy.least_used}>{t("policy.leastUsed")}</SquareMenuItem>
          </DenseSelect>
        </FormControl>
        <NoMarginHelperText>{t("policy.strategyDes")}</NoMarginHelperText>
      </SettingForm>
      <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
        {t("policy.create")}
      </Button>
    </form>
  );
};

export default LoadBalanceWizard;
