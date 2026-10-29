import { Box, Divider, IconButton, Link, Skeleton, Tooltip, Typography } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { deleteStoragePolicy, getStoragePolicyDetail } from "../../../api/api";
import { StoragePolicy } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { sizeToString } from "../../../util";
import { NoWrapBox, SquareChip } from "../../Common/StyledComponents";
import Delete from "../../Icons/Delete";
import Info from "../../Icons/Info";
import ArrowSync from "../../Icons/ArrowSync";
import { BorderedCardClickableBaImg } from "../Common/AdminCard";
import MigratePolicyDialog from "./MigratePolicyDialog";
import { PolicyPropsMap } from "./StoragePolicySetting";

export interface StoragePolicyCardProps {
  policy?: StoragePolicy;
  onRefresh?: () => void;
  loading?: boolean;
}

const StoragePolicyCard = ({ policy, onRefresh, loading }: StoragePolicyCardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [detail, setDetail] = useState<StoragePolicy | undefined>(undefined);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [migrateOpen, setMigrateOpen] = useState(false);
  const navigate = useNavigate();

  const loadDetail = useCallback(
    (e: React.MouseEvent<HTMLAnchorElement>) => {
      e.preventDefault();
      e.stopPropagation();
      if (policy) {
        setDetailLoading(true);
        dispatch(getStoragePolicyDetail(policy.id, true)).then((res) => {
          setDetail(res);
          setDetailLoading(false);
        });
      }
    },
    [policy, dispatch],
  );

  const handleDelete = useCallback(
    (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      dispatch(confirmOperation(t("policy.deletePolicyConfirmation", { name: policy?.name ?? "" }))).then(() => {
        setDeleteLoading(true);
        dispatch(deleteStoragePolicy(policy?.id ?? 0))
          .then(() => {
            onRefresh?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      });
    },
    [policy, dispatch, onRefresh],
  );

  // If loading is true, render a skeleton placeholder
  if (loading) {
    return (
      <Grid
        size={{
          xs: 12,
          md: 6,
          lg: 4,
        }}
      >
        <BorderedCardClickableBaImg img="/static/img/local.png">
          <Box
            sx={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <Typography variant="subtitle1" fontWeight={600}>
              <Skeleton variant="text" width={100} />
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <Skeleton variant="text" width={100} />
            </Typography>
          </Box>
          <NoWrapBox>
            <Skeleton variant="text" width={60} height={25} sx={{ mr: 1 }} />
          </NoWrapBox>
          <Divider sx={{ my: 1 }} />
          <Box
            sx={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <Skeleton variant="text" width="40%" height={20} />
            <Skeleton variant="circular" width={30} height={30} />
          </Box>
        </BorderedCardClickableBaImg>
      </Grid>
    );
  }

  return (
    <Grid
      size={{
        xs: 12,
        md: 6,
        lg: 4,
      }}
    >
      <BorderedCardClickableBaImg
        onClick={() => navigate(`/admin/policy/${policy?.id}`)}
        img={policy ? PolicyPropsMap[policy.type].img : undefined}
      >
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Typography variant="subtitle1" fontWeight={600}>
            {policy?.name}
          </Typography>
          {policy && (
            <Typography variant="body2" color="text.secondary">
              {t(PolicyPropsMap[policy.type].name)}
            </Typography>
          )}
        </Box>
        <NoWrapBox>
          {policy?.edges.groups?.map((group) => (
            <SquareChip sx={{ mr: 1 }} key={group.name} label={group.name} size="small" />
          ))}
          {!policy?.edges.groups && (
            <Box sx={{ display: "flex", alignItems: "center", minHeight: "25px" }} color={"text.secondary"}>
              <Info sx={{ mr: 0.5, fontSize: 20 }} />
              <Typography variant="caption">{t("policy.noGroupBinded")}</Typography>
            </Box>
          )}
        </NoWrapBox>
        <Divider sx={{ my: 1 }} />
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Typography variant="body2" color="text.secondary">
            {detailLoading ? (
              <Skeleton variant="text" width={30} />
            ) : detail ? (
              t("policy.policySummary", {
                count: detail.entities_count ?? 0,
                size: sizeToString(detail.entities_size ?? 0),
              })
            ) : (
              <Link underline="hover" href="#/" onClick={loadDetail}>
                {t("policy.loadSummary")}
              </Link>
            )}
          </Typography>

          <Box sx={{ display: "flex", gap: 0.5 }}>
            <Tooltip title={t("group.migratePolicy")}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  setMigrateOpen(true);
                }}
              >
                <ArrowSync fontSize="small" />
              </IconButton>
            </Tooltip>
            <IconButton size="small" onClick={handleDelete}>
              <Delete fontSize="small" />
            </IconButton>
          </Box>
        </Box>
      </BorderedCardClickableBaImg>

      <MigratePolicyDialog
        open={migrateOpen}
        sourcePolicy={policy ?? null}
        onClose={() => {
          setMigrateOpen(false);
          onRefresh?.();
        }}
      />
    </Grid>
  );
};

export default StoragePolicyCard;
