import { Box, debounce, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getShareDetail, getShareList } from "../../../api/api.ts";
import { Share } from "../../../api/dashboard.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DenseAutocomplete, DenseFilledTextField, SquareChip } from "../../Common/StyledComponents.tsx";

export interface ShareOption {
  /** Numeric share ID (admin API). */
  id: number;
  /** Hash ID of the share, used to build share URIs. */
  hash_id?: string;
  /** Display name, defaults to the shared file's name. */
  name?: string;
}

export interface SharesInputProps {
  value: ShareOption[];
  onChange: (value: ShareOption[]) => void;
}

const SharesInput = ({ value, onChange }: SharesInputProps) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [inputValue, setInputValue] = useState("");
  const [options, setOptions] = useState<ShareOption[]>([]);

  const toOption = (s: Share): ShareOption => ({
    id: s.id,
    hash_id: s.share_hash_id,
    name: s.edges?.file?.name ?? undefined,
  });

  // Exact numeric ID resolves that share; otherwise treat input as a
  // keyword and list recent shares as suggestions.
  const search = useMemo(
    () =>
      debounce((keyword: string) => {
        const numeric = parseInt(keyword);
        if (Number.isFinite(numeric) && String(numeric) === keyword.trim()) {
          dispatch(getShareDetail(numeric))
            .then((s) => setOptions((prev) => [toOption(s), ...prev.filter((p) => p.id !== s.id)].slice(0, 20)))
            .catch(() => {});
        } else {
          dispatch(getShareList({ page: 1, page_size: 10, order_by: "id", order_direction: "desc" }))
            .then((res) => setOptions(res.shares.map(toOption)))
            .catch(() => {});
        }
      }, 300),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [dispatch],
  );

  return (
    <DenseAutocomplete
      multiple
      options={options}
      value={value}
      inputValue={inputValue}
      blurOnSelect
      onInputChange={(_, v) => {
        setInputValue(v);
        if (v) {
          search(v);
        } else {
          setOptions([]);
        }
      }}
      isOptionEqualToValue={(a, b) => (a as ShareOption).id === (b as ShareOption).id}
      getOptionLabel={(o) => {
        const s = o as ShareOption;
        return s.name ? `${s.name} (#${s.id})` : `#${s.id}`;
      }}
      onChange={(_, next) => onChange(next as ShareOption[])}
      filterOptions={(x) => x as ShareOption[]}
      renderTags={(tagValue, getTagProps) =>
        (tagValue as ShareOption[]).map((option, index) => (
          <SquareChip
            {...getTagProps({ index })}
            key={option.id}
            label={option.name ? `${option.name} (#${option.id})` : `#${option.id}`}
          />
        ))
      }
      renderInput={(params) => (
        <DenseFilledTextField
          {...params}
          sx={{
            "& .MuiInputBase-root": {},
            "& .MuiInputBase-root.MuiOutlinedInput-root": {
              paddingTop: theme.spacing(0.6),
              paddingBottom: theme.spacing(0.6),
            },
            mt: 0,
          }}
          variant="outlined"
          margin="dense"
          placeholder={t("dashboard:settings.searchShare")}
          type="text"
          fullWidth
        />
      )}
      renderOption={(props, option) => {
        const s = option as ShareOption;
        return (
          <Box component="li" {...props} key={s.id}>
            {s.name ? `${s.name} (#${s.id})` : `#${s.id}`}
          </Box>
        );
      }}
    />
  );
};

export default SharesInput;
