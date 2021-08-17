import React, { ChangeEventHandler, useEffect, useState } from 'react';
import { PageProps } from "gatsby"
import { Typography, Grid, Form, Input, Button, Radio } from 'antd';
const { Title } = Typography;
const { TextArea } = Input
const { useBreakpoint } = Grid
import { SearchOutlined } from '@ant-design/icons';
import { injectIntl, WrappedComponentProps, FormattedMessage } from 'gatsby-plugin-intl';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { ExplorerQuery } from '~/components/ExplorerQuery'
import { titleStyles } from '~/styles';


// form props
interface ExplorerFormValues {
    emitterChain: number,
    emitterAddress: string,
    sequence: string
}
const formFields = ['emitterChain', 'emitterAddress', 'sequence']
const emitterChains = [
    { label: 'Solana', value: 1 },
    { label: 'Ethereum', value: 2 },
    { label: 'Terra', value: 3 },
    { label: 'Binance Smart Chain', value: 4 },

]

interface ExplorerProps extends PageProps, WrappedComponentProps<'intl'> { }
const Explorer = ({ location, intl, navigate }: ExplorerProps) => {

    const screens = useBreakpoint()
    const [, forceUpdate] = useState({});
    const [form] = Form.useForm<ExplorerFormValues>();
    const [emitterChain, setEmitterChain] = useState<ExplorerFormValues["emitterChain"]>()
    const [emitterAddress, setEmitterAddress] = useState<ExplorerFormValues["emitterAddress"]>()
    const [sequence, setSequence] = useState<ExplorerFormValues["sequence"]>()

    useEffect(() => {
        // To disable submit button on first load.
        forceUpdate({});
    }, [])

    useEffect(() => {

        if (location.search) {
            // take searchparams from the URL and set the values in the form
            const searchParams = new URLSearchParams(location.search);

            const chain = searchParams.get('emitterChain')
            const address = searchParams.get('emitterAddress')
            const sequence = searchParams.get('sequence')


            // get the current values from the form fields
            const { emitterChain, emitterAddress, sequence: seq } = form.getFieldsValue(true)

            // if the search params are different form values, update the form.
            if (chain) {
                if (Number(chain) !== emitterChain) {
                    form.setFieldsValue({ emitterChain: Number(chain) })
                }
                setEmitterChain(Number(chain))
            }
            if (address) {
                if (address !== emitterAddress) {
                    form.setFieldsValue({ emitterAddress: address })
                }
                setEmitterAddress(address)
            }
            if (sequence) {
                if (sequence !== seq) {
                    form.setFieldsValue({ sequence: sequence })
                }
                setSequence(sequence)
            }
        }
    }, [location.search])



    const onFinish = ({ emitterChain, emitterAddress, sequence }: ExplorerFormValues) => {
        // pushing to the history stack will cause the component to get new props, and useEffect will run.
        navigate(`/${intl.locale}/explorer/?emitterChain=${emitterChain}&emitterAddress=${emitterAddress}&sequence=${sequence}`)
    };

    const onAddress: ChangeEventHandler<HTMLTextAreaElement> = (e) => {
        if (e.currentTarget.value) {
            // trim whitespace
            form.setFieldsValue({ emitterAddress: e.currentTarget.value.replace(/\s/g, "") })
        }

    }
    const onSequence: ChangeEventHandler<HTMLInputElement> = (e) => {
        if (e.currentTarget.value) {
            // remove everything except numbers
            form.setFieldsValue({ sequence: e.currentTarget.value.replace(/\D/g, '') })
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
        <Layout>
            <SEO
                title={intl.formatMessage({ id: 'explorer.title' })}
                description={intl.formatMessage({ id: 'explorer.description' })}
            />
            <div
                className="center-content"
                style={{ paddingTop: screens.md === false ? 24 : 100 }}
            >
                <div
                    className="responsive-padding max-content-width"
                    style={{ width: '100%' }}
                >

                    <Title level={1} style={titleStyles}>{intl.formatMessage({ id: 'explorer.title' })}</Title>

                    <div>
                        <Form
                            layout="vertical"
                            form={form}
                            name="explorer-query"
                            onFinish={onFinish}
                            size="large"
                            style={{ width: '90%', maxWidth: 800, marginBlockEnd: 60, fontSize: 14 }}
                            colon={false}
                            requiredMark={false}
                            validateMessages={{ required: "'${label}' is required", }}
                        >
                            <Form.Item
                                name="emitterAddress"
                                label={formatLabel("explorer.emitterAddress")}
                                help={formatHelp("explorer.emitterAddressHelp")}
                                rules={[{ required: true }]}
                            >
                                <TextArea onChange={onAddress} allowClear autoSize />
                            </Form.Item>

                            <Form.Item
                                name="emitterChain"
                                label={formatLabel("explorer.emitterChain")}
                                help={formatHelp("explorer.emitterChainHelp")}
                                rules={[{ required: true }]}
                                style={
                                    screens.md === false ? {
                                        display: 'block', width: '100%'
                                    } : {
                                        display: 'inline-block', width: '50%'
                                    }}
                            >
                                <Radio.Group
                                    optionType="button"
                                    options={emitterChains}
                                />
                            </Form.Item>

                            <Form.Item shouldUpdate
                                style={
                                    screens.md === false ? {
                                        display: 'block', width: '100%'
                                    } : {
                                        display: 'inline-block', width: '50%'
                                    }}
                            >
                                {() => (

                                    <Form.Item
                                        name="sequence"
                                        label={formatLabel("explorer.sequence")}
                                        help={formatHelp("explorer.sequenceHelp")}
                                        rules={[{ required: true }]}
                                    >

                                        <Input
                                            onChange={onSequence}
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
                    {emitterChain && emitterAddress && sequence ? (
                        <ExplorerQuery emitterChain={emitterChain} emitterAddress={emitterAddress} sequence={sequence} />
                    ) : null}

                </div>
            </div>
        </Layout >
    )
};

export default injectIntl(Explorer)
