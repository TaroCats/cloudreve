import { Autocomplete, Typography } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getSearchUser, getUserInfo } from "../../../../api/api.ts";
import { User } from "../../../../api/user.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { NoLabelFilledTextField } from "../../../Common/StyledComponents.tsx";
import { PropsContentProps } from "./CustomPropsItem.tsx";

// UserPropsContent edits a "user" custom prop. The stored value is the
// target user's hash ID; the nickname is resolved for display.
const UserPropsContent = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [value, setValue] = useState(prop.value);
  const [inputValue, setInputValue] = useState("");
  const [options, setOptions] = useState<User[]>([]);
  const [resolved, setResolved] = useState<User | null>(null);

  useEffect(() => {
    setValue(prop.value);
  }, [prop.value]);

  // Resolve the stored hash ID to a user for display.
  useEffect(() => {
    if (!prop.value) {
      setResolved(null);
      return;
    }
    if (resolved?.id === prop.value) {
      return;
    }
    dispatch(getUserInfo(prop.value))
      .then((u) => setResolved(u))
      .catch(() => setResolved(null));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [prop.value]);

  // Debounced user search for the picker.
  useEffect(() => {
    if (!inputValue) {
      setOptions([]);
      return;
    }
    const timer = setTimeout(() => {
      dispatch(getSearchUser(inputValue))
        .then((users) => setOptions(users ?? []))
        .catch(() => setOptions([]));
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inputValue]);

  const selected = useMemo(() => {
    if (resolved) {
      return resolved;
    }
    return options.find((u) => u.id === value) ?? null;
  }, [resolved, options, value]);

  if (readOnly) {
    return <Typography variant="body2">{resolved?.nickname ?? value ?? "-"}</Typography>;
  }

  return (
    <Autocomplete
      size="small"
      fullWidth
      value={selected}
      options={options}
      loading={loading}
      disabled={loading}
      noOptionsText={t("application:fileManager.clickToEdit")}
      getOptionLabel={(u) => u.nickname ?? u.email ?? u.id}
      isOptionEqualToValue={(a, b) => a.id === b.id}
      onInputChange={(_, v) => setInputValue(v)}
      onChange={(_, u) => {
        const next = u?.id ?? "";
        if (next !== prop.value) {
          setValue(next);
          onChange(next);
        }
      }}
      onClick={(e) => e.stopPropagation()}
      renderInput={(params) => (
        <NoLabelFilledTextField
          {...params}
          variant="filled"
          fullSize={fullSize}
          placeholder={t("application:fileManager.clickToEdit")}
          fullWidth
        />
      )}
    />
  );
};

export default UserPropsContent;
