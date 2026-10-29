import { Delete } from "@mui/icons-material";
import {
  Box,
  Button,
  Checkbox,
  Chip,
  Container,
  Divider,
  Link,
  ListSubheader,
  MenuItem,
  Skeleton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { batchDeleteAuditLogs, getAuditLogs } from "../../../api/api";
import { AdminListService, AuditLog } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import {
  DenseFilledTextField,
  DenseSelect,
  NoWrapTableCell,
  NoWrapTypography,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../Common/StyledComponents";
import TimeBadge from "../../Common/TimeBadge";
import UserAvatar from "../../Common/User/UserAvatar";
import ArrowSync from "../../Icons/ArrowSync";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import UserDialog from "../User/UserDialog/UserDialog";
import { eventCategories, getEventCategory, getEventName } from "./eventTypes";

export const EventTypeQuery = "type";
export const EventUserQuery = "user";

const EventList = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, { defaultValue: "10" });
  const [type, setType] = useQueryState(EventTypeQuery, { defaultValue: "" });
  const [user, setUser] = useQueryState(EventUserQuery, { defaultValue: "" });
  const [userInput, setUserInput] = useState(user);

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);

  const pageInt = parseInt(page) || 1;
  const pageSizeInt = parseInt(pageSize) || 10;

  const fetchLogs = useCallback(() => {
    setLoading(true);
    setSelected([]);

    const params: AdminListService = {
      page: pageInt,
      page_size: pageSizeInt,
      order_by: "",
      order_direction: "desc",
      conditions: {
        event_type: type,
        event_user_id: user,
      },
    };

    dispatch(getAuditLogs(params))
      .then((res) => {
        setLogs(res.logs);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
      })
      .finally(() => {
        setLoading(false);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, type, user]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const applyUserFilter = () => {
    if (userInput !== user) {
      setUser(userInput);
      setPage("1");
    }
  };

  const handleDelete = () => {
    setDeleteLoading(true);
    dispatch(confirmOperation(t("event.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteAuditLogs(Array.from(selected)))
          .then(() => {
            fetchLogs();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      })
      .catch(() => {
        setDeleteLoading(false);
      });
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      setSelected(logs.map((n) => n.id));
      return;
    }
    setSelected([]);
  };

  const handleSelect = (id: number) => {
    const selectedIndex = selected.indexOf(id);
    if (selectedIndex === -1) {
      setSelected(selected.concat(id));
    } else {
      setSelected(selected.filter((v) => v !== id));
    }
  };

  const eventLabel = (typeValue: number) => {
    const name = getEventName(typeValue);
    return t(`settings.event.${name}`, { defaultValue: name });
  };

  return (
    <PageContainer>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <Container maxWidth="xl">
        <PageHeader title={t("nav.events")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }} alignItems="center" useFlexGap flexWrap="wrap">
          <DenseSelect
            value={type}
            onChange={(e) => {
              setType(String(e.target.value));
              setPage("1");
            }}
            size="small"
            sx={{ minWidth: 200 }}
            displayEmpty
          >
            <MenuItem value="">{t("event.allEventTypes")}</MenuItem>
            {eventCategories.map((category) => [
              <ListSubheader key={`${category.id}-header`}>{t(category.titleKey)}</ListSubheader>,
              ...category.types.map((value) => (
                <MenuItem key={value} value={String(value)}>
                  {eventLabel(value)}
                </MenuItem>
              )),
            ])}
          </DenseSelect>
          <DenseFilledTextField
            size="small"
            sx={{ width: 180 }}
            value={userInput}
            onChange={(e) => setUserInput(e.target.value)}
            onBlur={applyUserFilter}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                applyUserFilter();
              }
            }}
            placeholder={t("event.userID")}
          />
          <SecondaryButton onClick={fetchLogs} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>
          {selected.length > 0 && !deleteLoading && (
            <>
              <Divider orientation="vertical" flexItem />
              <Button
                startIcon={<Delete />}
                variant="contained"
                color="error"
                disabled={deleteLoading}
                onClick={handleDelete}
              >
                {t("event.deleteXEvents", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox" sx={{ width: "36px!important" }} width={50}>
                  <Checkbox
                    size="small"
                    indeterminate={selected.length > 0 && selected.length < logs.length}
                    checked={logs.length > 0 && selected.length === logs.length}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
                <NoWrapTableCell width={80}>{t("group.#")}</NoWrapTableCell>
                <NoWrapTableCell width={170}>{t("event.datetime")}</NoWrapTableCell>
                <NoWrapTableCell width={190}>{t("event.type")}</NoWrapTableCell>
                <NoWrapTableCell width={180}>{t("event.initiator")}</NoWrapTableCell>
                <NoWrapTableCell width={140}>{t("event.ip")}</NoWrapTableCell>
                <TableCell>{t("event.rawContent")}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                logs.map((log) => (
                  <TableRow hover key={log.id} selected={selected.indexOf(log.id) !== -1}>
                    <TableCell padding="checkbox">
                      <Checkbox
                        size="small"
                        disableRipple
                        color="primary"
                        disabled={deleteLoading}
                        checked={selected.indexOf(log.id) !== -1}
                        onClick={() => handleSelect(log.id)}
                      />
                    </TableCell>
                    <NoWrapTableCell>
                      <NoWrapTypography variant="inherit">{log.id}</NoWrapTypography>
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <TimeBadge datetime={log.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <Chip
                        size="small"
                        variant="outlined"
                        color={getEventCategory(log.type ?? 0)?.color ?? "default"}
                        label={eventLabel(log.type ?? 0)}
                      />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      {log.edges?.user ? (
                        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                          <UserAvatar
                            sx={{ width: 24, height: 24 }}
                            overwriteTextSize
                            user={{
                              id: log.user_hash_id ?? "",
                              nickname: log.user_nick ?? log.edges.user.nick ?? "",
                              created_at: log.edges.user.created_at ?? "",
                            }}
                          />
                          <NoWrapTypography variant="inherit">
                            <Link
                              href="#/"
                              underline="hover"
                              onClick={(e: React.MouseEvent<HTMLElement>) => {
                                e.preventDefault();
                                setUserDialogID(log.edges?.user?.id ?? 0);
                                setUserDialogOpen(true);
                              }}
                              sx={{
                                whiteSpace: "nowrap",
                                overflow: "hidden",
                                textOverflow: "ellipsis",
                              }}
                            >
                              {log.user_nick ?? log.edges.user.nick}
                            </Link>
                          </NoWrapTypography>
                        </Box>
                      ) : (
                        "-"
                      )}
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <NoWrapTypography variant="inherit">{log.ip || "-"}</NoWrapTypography>
                    </NoWrapTableCell>
                    <TableCell>
                      {log.detail && Object.keys(log.detail).length > 0 ? (
                        <Tooltip
                          title={
                            <Box
                              component="pre"
                              sx={{ m: 0, whiteSpace: "pre-wrap", wordBreak: "break-all", fontSize: 12 }}
                            >
                              {JSON.stringify(log.detail, null, 2)}
                            </Box>
                          }
                        >
                          <NoWrapTypography variant="inherit" sx={{ maxWidth: 320, display: "block" }}>
                            {Object.entries(log.detail)
                              .map(([k, v]) => `${k}: ${v}`)
                              .join(", ")}
                          </NoWrapTypography>
                        </Tooltip>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              {loading &&
                Array.from(Array(10)).map((_, index) => (
                  <TableRow key={`loading-${index}`} sx={{ height: "43px" }}>
                    <TableCell padding="checkbox">
                      <Skeleton variant="circular" width={24} height={24} />
                    </TableCell>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={30} />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={130} />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={120} />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={120} />
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={90} />
                    </NoWrapTableCell>
                    <TableCell>
                      <Skeleton variant="text" width={220} />
                    </TableCell>
                  </TableRow>
                ))}
              {!loading && logs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} align="center">
                    <Typography color="text.secondary" sx={{ py: 4 }}>
                      {t("event.empty")}
                    </Typography>
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

export default EventList;
