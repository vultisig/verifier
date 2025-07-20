import { List, Modal } from "antd";
import { useApp } from "hooks/useApp";
import { useGoBack } from "hooks/useGoBack";
import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { modalHash } from "utils/constants/core";
import { Language, languageNames, languages } from "utils/constants/language";

export const LanguageModal = () => {
  const [visible, setVisible] = useState(false);
  const { setLanguage } = useApp();
  const { hash } = useLocation();
  const goBack = useGoBack();

  const handleSelect = (key: Language): void => {
    setLanguage(key);

    goBack();
  };

  useEffect(() => setVisible(hash === modalHash.language), [hash]);

  return (
    <Modal
      centered={true}
      footer={false}
      maskClosable={false}
      onCancel={() => goBack()}
      open={visible}
      styles={{ footer: { display: "none" } }}
      title="Change Language"
      width={360}
    >
      <List
        dataSource={languages.map((key) => ({
          key,
          title: languageNames[key],
        }))}
        renderItem={({ key, title }) => (
          <List.Item onClick={() => handleSelect(key)}>{title}</List.Item>
        )}
      />
    </Modal>
  );
};
