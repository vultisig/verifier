import { List, Modal } from "antd";
import { useApp } from "hooks/useApp";
import { useGoBack } from "hooks/useGoBack";
import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { useTheme } from "styled-components";
import { modalHash } from "utils/constants/core";
import {
  currencies,
  Currency,
  currencySymbols,
} from "utils/constants/currency";

import { Stack } from "./Stack";

export const CurrencyModal = () => {
  const [visible, setVisible] = useState(false);
  const { currency, setCurrency } = useApp();
  const { hash } = useLocation();
  const goBack = useGoBack();
  const colors = useTheme();

  const handleSelect = (key: Currency): void => {
    setCurrency(key);

    goBack();
  };

  useEffect(() => setVisible(hash === modalHash.currency), [hash]);

  return (
    <Modal
      centered={true}
      footer={false}
      maskClosable={false}
      onCancel={() => goBack()}
      open={visible}
      styles={{ footer: { display: "none" } }}
      title="Change Currency"
      width={360}
    >
      <List
        dataSource={currencies.map((key) => ({
          key,
          title: currencySymbols[key],
        }))}
        renderItem={({ key, title }) => {
          const isActive = key === currency;

          return (
            <Stack
              as={List.Item}
              key={key}
              onClick={() => handleSelect(key)}
              {...(isActive
                ? {
                    $style: { color: `${colors.success.toHex()} !important` },
                  }
                : {
                    $hover: { color: colors.buttonPrimary.toHex() },
                    $style: { cursor: "pointer" },
                  })}
            >
              <span>{title}</span>
              <span>{key.toUpperCase()}</span>
            </Stack>
          );
        }}
      />
    </Modal>
  );
};
