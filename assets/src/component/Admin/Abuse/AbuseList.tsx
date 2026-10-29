import { Delete } from "@mui/icons-material";
import {
  Box,
  Button,
  Checkbox,
  Container,
  Divider,
  IconButton,
  Link,
  Skeleton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { batchDeleteAbuseReports, getAbuseReports } from "../../../api/api";
import { AbuseReport, AdminListService } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import {
  NoWrapTableCell,
  NoWrapTypography,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../Common/StyledComponents";
import TimeBadge from "../../Common/TimeBadge";
import UserAvatar from "../../Common/User/UserAvatar";
import FileTypeIcon from "../../FileManager/Explorer/FileTypeIcon";
import ArrowSync from "../../Icons/ArrowSync";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";

const columnCount = 8;

interface AbuseReportRowProps {
  report?: AbuseReport;
  loading?: boolean;
  deleting?: boolean;
  selected?: boolean;
  onSelect?: (id: number) => void;
  onDismiss?: () => void;
}

const AbuseReportRow = ({ report, loading, deleting, selected, onSelect, onDismiss }: AbuseReportRowProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [dismissLoading, setDismissLoading] = useState(false);

  const onSelectClick = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onSelect?.(report?.id ?? 0);
  };

  const onDismissClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    dispatch(confirmOperation(t("abuseReport.confirmDelete"))).then(() => {
      if (report?.id) {
        setDismissLoading(true);
        dispatch(batchDeleteAbuseReports([report.id]))
          .then(() => {
            onDismiss?.();
          })
          .finally(() => {
            setDismissLoading(false);
          });
      }
    });
  };

  const onOpenLink = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    window.open(report?.share_link ?? "", "_blank");
  };

  if (loading) {
    return (
      <TableRow sx={{ height: "43px" }}>
        <TableCell padding="checkbox">
          <Skeleton variant="circular" width={24} height={24} />
        </TableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={30} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Skeleton variant="circular" width={24} height={24} />
            <Skeleton variant="text" width={160} />
          </Box>
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Skeleton variant="circular" width={24} height={24} />
            <Skeleton variant="text" width={60} />
          </Box>
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={60} />
        </NoWrapTableCell>
        <TableCell>
          <Skeleton variant="text" width={180} />
        </TableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={70} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
      </TableRow>
    );
  }

  const share = report?.edges?.share;

  return (
    <TableRow hover key={report?.id} selected={selected}>
      <TableCell padding="checkbox">
        <Checkbox
          disabled={deleting}
          size="small"
          disableRipple
          color="primary"
          onClick={onSelectClick}
          checked={selected}
        />
      </TableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{report?.id}</NoWrapTypography>
      </NoWrapTableCell>
      <TableCell>
        {share ? (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <FileTypeIcon name={report?.file_name ?? ""} fileType={report?.edges?.share?.edges?.file?.type ?? 0} />
            <NoWrapTypography variant="inherit">
              {report?.file_name ? (
                <Link href={report?.share_link ?? "#"} target="_blank" rel="noopener" underline="hover">
                  {report?.file_name}
                </Link>
              ) : (
                <Trans
                  i18nKey="abuseReport.shareLink"
                  values={{ id: report?.id }}
                  components={[<Link onClick={onOpenLink} underline="hover" href="#/" />]}
                />
              )}
            </NoWrapTypography>
          </Box>
        ) : (
          <em>{t("abuseReport.deletedShare")}</em>
        )}
      </TableCell>
      <NoWrapTableCell>
        {report?.owner_name ? (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <UserAvatar
              sx={{ width: 24, height: 24 }}
              overwriteTextSize
              user={{
                id: report?.owner_hash_id ?? "",
                nickname: report?.owner_name ?? "",
                created_at: report?.edges?.share?.edges?.user?.created_at ?? "",
              }}
            />
            <NoWrapTypography variant="inherit">{report?.owner_name}</NoWrapTypography>
          </Box>
        ) : (
          <em>{share ? t("abuseReport.deletedUser") : "-"}</em>
        )}
      </NoWrapTableCell>
      <NoWrapTableCell>
        {report?.reporter_name ? (
          <NoWrapTypography variant="inherit">{report?.reporter_name}</NoWrapTypography>
        ) : report?.reporter_id ? (
          <em>{t("abuseReport.deletedUser")}</em>
        ) : (
          <em>{t("abuseReport.anonymous")}</em>
        )}
      </NoWrapTableCell>
      <TableCell>
        <NoWrapTypography variant="inherit" title={report?.reason}>
          {report?.reason}
        </NoWrapTypography>
      </TableCell>
      <NoWrapTableCell>
        <TimeBadge datetime={report?.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
      </NoWrapTableCell>
      <NoWrapTableCell>
        <IconButton size="small" onClick={onDismissClick} disabled={dismissLoading || deleting}>
          <Delete fontSize="small" />
        </IconButton>
      </NoWrapTableCell>
    </TableRow>
  );
};

/** AbuseList renders filed abuse reports and allows dismissing them. */
const AbuseList = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [reports, setReports] = useState<AbuseReport[]>([]);
  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, { defaultValue: "10" });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, { defaultValue: "" });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });

  const pageInt = parseInt(page) || 1;
  const pageSizeInt = parseInt(pageSize) || 10;

  useEffect(() => {
    fetchReports();
  }, [page, pageSize, orderBy, orderDirection]);

  const fetchReports = () => {
    setLoading(true);
    setSelected([]);

    const params: AdminListService = {
      page: pageInt,
      page_size: pageSizeInt,
      order_by: orderBy ?? "",
      order_direction: orderDirection ?? "desc",
    };

    dispatch(getAbuseReports(params))
      .then((res) => {
        setReports(res.reports ?? []);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const handleDismissSelected = () => {
    setDeleteLoading(true);
    dispatch(confirmOperation(t("abuseReport.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteAbuseReports(Array.from(selected)))
          .then(() => {
            fetchReports();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      })
      .finally(() => {
        setDeleteLoading(false);
      });
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      setSelected(reports.map((n) => n.id));
      return;
    }
    setSelected([]);
  };

  const handleSelect = useCallback(
    (id: number) => {
      const selectedIndex = selected.indexOf(id);
      let newSelected: readonly number[] = [];

      if (selectedIndex === -1) {
        newSelected = newSelected.concat(selected, id);
      } else if (selectedIndex === 0) {
        newSelected = newSelected.concat(selected.slice(1));
      } else if (selectedIndex === selected.length - 1) {
        newSelected = newSelected.concat(selected.slice(0, -1));
      } else if (selectedIndex > 0) {
        newSelected = newSelected.concat(selected.slice(0, selectedIndex), selected.slice(selectedIndex + 1));
      }
      setSelected(newSelected);
    },
    [selected],
  );

  const orderById = orderBy === "id" || orderBy === "";
  const direction = orderDirection as "asc" | "desc";
  const onSortClick = (field: string) => () => {
    const alreadySorted = orderBy === field || (field === "id" && orderById);
    setOrderBy(field);
    setOrderDirection(alreadySorted ? (direction === "asc" ? "desc" : "asc") : "asc");
  };

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("nav.abuseReport")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <SecondaryButton onClick={fetchReports} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>
          {selected.length > 0 && !isMobile && (
            <>
              <Divider orientation="vertical" flexItem />
              <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDismissSelected}>
                {t("abuseReport.deleteXAbuseReports", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDismissSelected}>
              {t("abuseReport.deleteXAbuseReports", { num: selected.length })}
            </Button>
          </Stack>
        )}
        <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox" sx={{ width: "36px!important" }} width={50}>
                  <Checkbox
                    size="small"
                    indeterminate={selected.length > 0 && selected.length < reports.length}
                    checked={reports.length > 0 && selected.length === reports.length}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
                <NoWrapTableCell width={80}>
                  <TableSortLabel
                    active={orderById}
                    direction={orderById ? direction : "asc"}
                    onClick={onSortClick("id")}
                  >
                    {t("group.#")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={300}>{t("abuseReport.share")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("abuseReport.reportedUser")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("abuseReport.reporter")}</NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("abuseReport.reason")}</NoWrapTableCell>
                <NoWrapTableCell width={180}>
                  <TableSortLabel
                    active={orderBy === "created_at"}
                    direction={orderBy === "created_at" ? direction : "asc"}
                    onClick={onSortClick("created_at")}
                  >
                    {t("abuseReport.createdAt")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={80}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                reports.map((report) => (
                  <AbuseReportRow
                    deleting={deleteLoading}
                    key={report.id}
                    report={report}
                    onDismiss={fetchReports}
                    selected={selected.indexOf(report.id) !== -1}
                    onSelect={handleSelect}
                  />
                ))}
              {loading &&
                Array.from(Array(reports.length > 0 ? Math.min(reports.length, 10) : 10)).map((_, index) => (
                  <AbuseReportRow key={`loading-${index}`} loading={true} />
                ))}
              {!loading && reports.length === 0 && (
                <TableRow>
                  <TableCell colSpan={columnCount} align="center">
                    <NoWrapTypography color="text.secondary" sx={{ py: 4 }}>
                      {t("abuseReport.empty")}
                    </NoWrapTypography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[10, 25, 50, 100, 200, 500]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default AbuseList;
