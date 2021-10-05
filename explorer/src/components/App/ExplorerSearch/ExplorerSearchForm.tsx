import React, { ChangeEventHandler, useEffect, useState } from 'react';
import { PageProps } from "gatsby"
import { Grid, Form, Input, Button, Radio, RadioChangeEvent } from 'antd';
const { TextArea } = Input
const { useBreakpoint } = Grid
import { SearchOutlined } from '@ant-design/icons';
import { FormattedMessage, useIntl } from 'gatsby-plugin-intl';


import { ExplorerQuery } from '~/components/ExplorerQuery'
import { ChainID, chainIDs } from '~/utils/misc/constants';


// form props
interface ExplorerFormValues {
    emitterChain: number,
    emitterAddress: string,
    sequence: string
}
const formFields = ['emitterChain', 'emitterAddress', 'sequence']
const emitterChains = [
    { label: ChainID[1], value: chainIDs['solana'] },
    { label: ChainID[2], value: chainIDs['ethereum'] },
    { label: ChainID[3], value: chainIDs['terra'] },
    { label: ChainID[4], value: chainIDs['bsc'] },
    { label: ChainID[5], value: chainIDs['polygon'] },

]

interface ExplorerSearchProps {
    location: PageProps["location"],
    navigate: PageProps["navigate"]
}
const ExplorerSearchForm: React.FC<ExplorerSearchProps> = ({ location, navigate }) => {
    const intl = useIntl()
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
            const seqQuery = searchParams.get('sequence')

            // get the current values from the form fields
            const { emitterChain, emitterAddress, sequence: seqForm } = form.getFieldsValue(true)

            // if the search params are different form values, update the form.
            if (Number(chain) !== emitterChain) {
                form.setFieldsValue({ emitterChain: Number(chain) })
            }
            setEmitterChain(Number(chain))

            if (address !== emitterAddress) {
                form.setFieldsValue({ emitterAddress: address || undefined })
            }
            setEmitterAddress(address || undefined)

            if (seqQuery !== seqForm) {
                form.setFieldsValue({ sequence: seqQuery || undefined })
            }
            setSequence(seqQuery || undefined)
        } else {
            // clear state
            setEmitterChain(undefined)
            setEmitterAddress(undefined)
            setSequence(undefined)
        }
    }, [location.search])



    const onFinish = ({ emitterChain, emitterAddress, sequence }: ExplorerFormValues) => {
        // pushing to the history stack will cause the component to get new props, and useEffect will run.
        navigate(`/${intl.locale}/explorer/?emitterChain=${emitterChain}&emitterAddress=${emitterAddress}&sequence=${sequence}`)
    };

    const onChain = (e: RadioChangeEvent) => {
        if (e.target.value) {
            setEmitterChain(e.target.value)
        }
    }
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
        <>

            <div style={{ display: 'flex', justifyContent: 'center', width: '100%' }}>
                <Form
                    layout="vertical"
                    form={form}
                    name="explorer-message-query"
                    onFinish={onFinish}
                    size="large"
                    style={{ width: '90%', maxWidth: 800, fontSize: 14 }}
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
                                display: 'inline-block', width: '60%'
                            }}
                    >
                        <Radio.Group
                            optionType="button"
                            options={emitterChains}
                            onChange={onChain}
                        />
                    </Form.Item>

                    <Form.Item shouldUpdate
                        style={
                            screens.md === false ? {
                                display: 'block', width: '100%'
                            } : {
                                display: 'inline-block', width: '40%'
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
        </ >
    )
};

export default ExplorerSearchForm
