import React from 'react';
import { Modal, Typography, Tag, Tabs, TabPane } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';

const InputOutputModal = ({
  showInputOutput,
  setShowInputOutput,
  inputOutputData,
  copyText,
  t,
}) => {
  if (!inputOutputData) return null;

  const { requestBody, responseBody, modelName } = inputOutputData;

  const formatJson = (str) => {
    if (!str) return '';
    try {
      return JSON.stringify(JSON.parse(str), null, 2);
    } catch {
      return str;
    }
  };

  const renderContent = (content, title) => {
    if (!content) {
      return (
        <div style={{ padding: 20, textAlign: 'center', color: 'var(--semi-color-text-2)' }}>
          {t('暂无数据')}
        </div>
      );
    }

    return (
      <div style={{ position: 'relative' }}>
        <div
          style={{
            position: 'absolute',
            top: 8,
            right: 8,
            zIndex: 1,
            cursor: 'pointer',
          }}
          onClick={(e) => copyText(e, content)}
        >
          <IconCopy style={{ color: 'var(--semi-color-text-2)' }} />
        </div>
        <pre
          style={{
            background: 'var(--semi-color-fill-0)',
            borderRadius: 8,
            padding: 16,
            margin: 0,
            maxHeight: 400,
            overflow: 'auto',
            fontSize: 13,
            lineHeight: 1.6,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
          }}
        >
          {formatJson(content)}
        </pre>
      </div>
    );
  };

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <span>{t('输入输出详情')}</span>
          {modelName && <Tag color='blue' shape='circle'>{modelName}</Tag>}
        </div>
      }
      visible={showInputOutput}
      onCancel={() => setShowInputOutput(false)}
      footer={null}
      centered
      closable
      maskClosable
      width={800}
    >
      <div style={{ padding: '12px 0' }}>
        <Tabs type='line' defaultActiveKey='request'>
          <TabPane tab={t('请求体')} itemKey='request'>
            {renderContent(requestBody, t('请求体'))}
          </TabPane>
          <TabPane tab={t('响应体')} itemKey='response'>
            {renderContent(responseBody, t('响应体'))}
          </TabPane>
        </Tabs>
      </div>
    </Modal>
  );
};

export default InputOutputModal;
