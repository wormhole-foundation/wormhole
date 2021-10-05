import React, { ChangeEventHandler, useEffect, useState } from 'react';
import { PageProps } from "gatsby"
import { Grid, Form, Input, Button, } from 'antd';
const { useBreakpoint } = Grid
import { SearchOutlined } from '@ant-design/icons';
import { FormattedMessage, useIntl } from 'gatsby-plugin-intl';

import { ExplorerQuery } from '~/components/ExplorerQuery'


// form props
interface ExplorerTxValues {
    txId: string,
}
const formFields = ['txId']

interface ExplorerSearchProps {
    location: PageProps["location"],
    navigate: PageProps["navigate"]
}
const ExplorerTxForm: React.FC<ExplorerSearchProps> = ({ location, navigate }) => {
    const intl = useIntl()
    const screens = useBreakpoint()
    const [, forceUpdate] = useState({});
    const [form] = Form.useForm<ExplorerTxValues>();
    const [txId, setTxId] = useState<ExplorerTxValues["txId"]>()

    useEffect(() => {
        // To disable submit button on first load.
        forceUpdate({});
    }, [])

    useEffect(() => {

        if (location.search) {
            // take searchparams from the URL and set the values in the form
            const searchParams = new URLSearchParams(location.search);

            const txQuery = searchParams.get('txId')

            // get the current values from the form fields
            const { txId: txForm } = form.getFieldsValue(true)

            // if the search params are different form values, update the form.
            if (txQuery) {
                if (txQuery !== txForm) {
                    form.setFieldsValue({ txId: txQuery })
                }
                setTxId(txQuery)
            }
        } else {
            // clear state
            setTxId(undefined)
        }
    }, [location.search])



    const onFinish = ({ txId }: ExplorerTxValues) => {
        // pushing to the history stack will cause the component to get new props, and useEffect will run.
        navigate(`/${intl.locale}/explorer/?txId=${txId}`)
    };

    const onTxId: ChangeEventHandler<HTMLInputElement> = (e) => {
        if (e.currentTarget.value) {
            // trim whitespace
            form.setFieldsValue({ txId: e.currentTarget.value.replace(/\s/g, "") })
        }
    }
    const formatLabel = (textKey: string) => (
        <span style={{ fontSize: 16 }}>
            <FormattedMessage id={textKey} />
        </span>

    )
    const formatHelp = (textKey: string) => (
        <span style={{ fontSize: 14 }}>
            <FormattedMessage id={textKey} />
        </span>
    )


    return (
        <>

            <div style={{ display: 'flex', justifyContent: 'center' }}>
                <Form
                    layout="vertical"
                    form={form}
                    name="explorer-tx-query"
                    onFinish={onFinish}
                    size="large"
                    style={{ width: '90%', maxWidth: 800, fontSize: 14 }}
                    colon={false}
                    requiredMark={false}
                    validateMessages={{ required: "'${label}' is required", }}
                >

                    <Form.Item shouldUpdate
                        style={
                            screens.md === false ? {
                                display: 'block', width: '100%'
                            } : {
                                display: 'inline-block', width: '100%'
                            }}
                    >
                        {() => (

                            <Form.Item
                                name="txId"
                                label={formatLabel("explorer.txId")}
                                help={formatHelp("explorer.txIdHelp")}
                                rules={[{ required: true }]}
                            >

                                <Input
                                    onChange={onTxId}
                                    style={{ padding: "0 0 0 14px" }}
                                    allowClear
                                    suffix={
                                        <Button
                                            size="large"
                                            type="primary"
                                            style={{ width: 80 }}
                                            icon={
                                                <SearchOutlined style={{ fontSize: 16, color: 'black' }} />
                                            }
                                            htmlType="submit"
                                            disabled={
                                                // true if the value of any field is falsey, or
                                                (Object.values({ ...form.getFieldsValue(formFields) }).some(v => !v)) ||
                                                // true if the length of the errors array is true.
                                                !!form.getFieldsError().filter(({ errors }) => errors.length).length
                                            }
                                        />
                                    }
                                />

                            </Form.Item>
                        )}
                    </Form.Item>

                </Form>
            </div>
            {txId ? (
                <ExplorerQuery txId={txId} />
            ) : null}
        </ >
    )
};

export default ExplorerTxForm
