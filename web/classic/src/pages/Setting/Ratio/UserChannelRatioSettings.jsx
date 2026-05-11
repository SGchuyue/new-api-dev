/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Form,
  Modal,
  Popconfirm,
  Space,
  Spin,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDelete, IconEdit } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function UserChannelRatioSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [ratios, setRatios] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState(null);
  const [users, setUsers] = useState([]);
  const [channels, setChannels] = useState([]);
  const formApiRef = useRef(null);

  const loadData = async (p = page, ps = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/user_channel_ratio/?p=${p}&page_size=${ps}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        setRatios(data.items || []);
        setTotal(data.total || 0);
        setPage(data.page || p);
      } else {
        showError(message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  const fetchUsers = async () => {
    try {
      const res = await API.get('/api/user/?p=0&page_size=9999');
      if (res.data.success) {
        setUsers(
          res.data.data.items.map((u) => ({
            label: `${u.username} (ID:${u.id})`,
            value: u.id,
          })),
        );
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const fetchChannels = async () => {
    try {
      const res = await API.get('/api/channel/?p=0&page_size=9999');
      if (res.data.success) {
        setChannels(
          res.data.data.items.map((c) => ({
            label: `${c.name} (ID:${c.id})`,
            value: c.id,
          })),
        );
      }
    } catch (e) {
      showError(e.message);
    }
  };

  useEffect(() => {
    loadData();
    fetchUsers();
    fetchChannels();
  }, []);

  const handleAdd = () => {
    setEditingRecord(null);
    setModalVisible(true);
    setTimeout(() => {
      formApiRef.current?.setValues({
        user_id: undefined,
        channel_id: undefined,
        ratio: 1,
      });
    }, 100);
  };

  const handleEdit = (record) => {
    setEditingRecord(record);
    setModalVisible(true);
    setTimeout(() => {
      formApiRef.current?.setValues({
        user_id: record.user_id,
        channel_id: record.channel_id,
        ratio: record.ratio,
      });
    }, 100);
  };

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/user_channel_ratio/${id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const handleSubmit = async (values) => {
    setLoading(true);
    try {
      let res;
      if (editingRecord) {
        res = await API.put('/api/user_channel_ratio/', {
          id: editingRecord.id,
          user_id: values.user_id,
          channel_id: values.channel_id,
          ratio: values.ratio,
        });
      } else {
        res = await API.post('/api/user_channel_ratio/', {
          user_id: values.user_id,
          channel_id: values.channel_id,
          ratio: values.ratio,
        });
      }
      if (res.data.success) {
        showSuccess(editingRecord ? t('更新成功') : t('创建成功'));
        setModalVisible(false);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  const getUsername = (userId) => {
    const user = users.find((u) => u.value === userId);
    return user ? user.label : `ID: ${userId}`;
  };

  const getChannelName = (channelId) => {
    const channel = channels.find((c) => c.value === channelId);
    return channel ? channel.label : `ID: ${channelId}`;
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('用户'),
      dataIndex: 'user_id',
      render: (text) => <Tag color='blue'>{getUsername(text)}</Tag>,
    },
    {
      title: t('渠道'),
      dataIndex: 'channel_id',
      render: (text) => <Tag color='green'>{getChannelName(text)}</Tag>,
    },
    {
      title: t('倍率'),
      dataIndex: 'ratio',
      render: (text) => (
        <Tag color={text < 1 ? 'orange' : text > 1 ? 'red' : 'grey'}>
          {text}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconEdit />}
            size='small'
            onClick={() => handleEdit(record)}
          >
            {t('编辑')}
          </Button>
          <Popconfirm
            title={t('确定删除？')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button icon={<IconDelete />} size='small' type='danger'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 16 }}>
        <Button icon={<IconPlus />} theme='solid' onClick={handleAdd}>
          {t('新增用户倍率')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={ratios}
        rowKey='id'
        pagination={{
          currentPage: page,
          pageSize: pageSize,
          total: total,
          onPageChange: (p) => {
            setPage(p);
            loadData(p, pageSize);
          },
          onPageSizeChange: (ps) => {
            setPageSize(ps);
            loadData(1, ps);
          },
        }}
      />

      <Modal
        title={editingRecord ? t('编辑用户倍率') : t('新增用户倍率')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => formApiRef.current?.submitForm()}
        centered
      >
        <Form
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={handleSubmit}
        >
          <Form.Select
            field='user_id'
            label={t('用户')}
            placeholder={t('请选择用户')}
            optionList={users}
            filter
            rules={[{ required: true, message: t('请选择用户') }]}
            style={{ width: '100%' }}
          />
          <Form.Select
            field='channel_id'
            label={t('渠道')}
            placeholder={t('请选择渠道')}
            optionList={channels}
            filter
            rules={[{ required: true, message: t('请选择渠道') }]}
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field='ratio'
            label={t('倍率')}
            placeholder={t('请输入倍率')}
            step={0.1}
            min={0}
            rules={[{ required: true, message: t('请输入倍率') }]}
            style={{ width: '100%' }}
          />
        </Form>
      </Modal>
    </Spin>
  );
}
