import Button from "@/modules/core/components/ui/button/Button";
import { useNavigate, useParams } from "react-router-dom";
import ChevronLeft from "@/assets/ChevronLeft.svg?react";
import logo from "../../../../assets/DCA-image.png"; // todo hardcoded until this image is stored in DB
import "./PluginDetail.css";
import { useEffect, useRef, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { Plugin, PluginPricing } from "../../models/plugin";
import Reviews from "@/modules/review/components/reviews/Reviews";
import { publish } from "@/utils/eventBus";
import { ReviewProvider } from "@/modules/review/context/ReviewProvider";
import VulticonnectWalletService from "@/modules/shared/wallet/vulticonnectWalletService";
import RecipeSchema from "@/modules/plugin/components/recipe_schema/recipe_Schema";
import { useWallet } from "@/modules/shared/wallet/WalletProvider";
import PolicyTable from "../../../policy/policy-table/PolicyTable";
import Modal from "@/modules/core/components/ui/modal/Modal";


const PluginDetail = () => {
    const navigate = useNavigate();
    const [plugin, setPlugin] = useState<Plugin | null>(null);
    const [isInstalled, setIsInstalled] = useState<boolean>(false);
    const [uninstallModalOpen, setUninstallModalOpen] = useState<boolean>(false);
    const [showRecipeSchema, setShowRecipeSchema] = useState(false);
    const { isConnected, connect, vault } = useWallet();
    const { pluginId } = useParams<{ pluginId: string }>();

    const checkPluginInstalled = async () => {
        if (isConnected && pluginId && vault?.publicKeyEcdsa) {
            const isInstalled = await MarketplaceService.isPluginInstalled(pluginId, vault?.publicKeyEcdsa);

            setIsInstalled(isInstalled);
        } else {
            setIsInstalled(false);
        }
    };

    const uninstallPlugin = async () => {
        try {
            await MarketplaceService.uninstallPlugin(pluginId!);
            setIsInstalled(false);
            publish("onToast", {
                message: "Plugin uninstalled successfully",
                type: "success",
            });
        } catch (error) {
            console.error("Failed to uninstall plugin:", error);
            publish("onToast", {
                message: "Failed to uninstall plugin",
                type: "error",
            });
        }
    };

    const fetchPlugin = async () => {
        if (pluginId) {
            try {
                const fetchedPlugin = await MarketplaceService.getPlugin(pluginId);
                setPlugin(fetchedPlugin);
            } catch (error) {
                if (error instanceof Error) {
                    console.error("Failed to get plugin:", error.message);
                    publish("onToast", {
                        message: "Failed to get plugin",
                        type: "error",
                    });
                }
            }
        }
    };

    const pricingText = (pricing?: PluginPricing) => {
        if (!pricing) {
            return "This plugin is free";
        }
        switch (pricing.type) {
            case "once":
                return `\$${pricing.amount / 1e6} one off installation fee`;
            case "recurring":
                return `\$${pricing.amount / 1e6} ${pricing.frequency} recurring fee`;
            case "per-tx":
                return `\$${pricing.amount / 1e6} per transaction fee`;
            default:
                return "Unknown pricing type";
        }
    };

    useEffect(() => {
        checkPluginInstalled();
    }, [isConnected, pluginId, vault]);

    useEffect(() => {
        fetchPlugin();
    }, [pluginId]);
    const checkIntervalRef = useRef<NodeJS.Timeout>();
    const checkTimeoutRef = useRef<NodeJS.Timeout>();

    const handleStartReshare = async () => {
        try {
            await VulticonnectWalletService.startReshareSession(pluginId);
        } catch (err) {
            console.error("Failed to start reshare session", err);
        }

        // Start checking every second
        checkIntervalRef.current = setInterval(async () => {
            await checkPluginInstalled();
        }, 2000);

        // Timeout after 5 minutes
        checkTimeoutRef.current = setTimeout(
            () => {
                if (checkIntervalRef.current) {
                    clearInterval(checkIntervalRef.current);
                }
                console.warn("Plugin install check timed out after 5 minutes");
            },
            5 * 60 * 1000
        ); // 5 minutes
    };

    // stop checking when isInstalled becomes true
    useEffect(() => {
        if (isInstalled && checkIntervalRef.current) {
            clearInterval(checkIntervalRef.current);
            if (checkTimeoutRef.current) {
                clearTimeout(checkTimeoutRef.current);
            }
        }
    }, [isInstalled]);

    useEffect(() => {
        return () => {
            if (checkIntervalRef.current) clearInterval(checkIntervalRef.current);
            if (checkTimeoutRef.current) clearTimeout(checkTimeoutRef.current);
        };
    }, []);

    return (
        <>
            <div className="only-section plugin-detail">
                <Button
                    size="small"
                    type="button"
                    style={{ paddingLeft: "0px", paddingTop: "2rem" }}
                    styleType="tertiary"
                    onClick={() => navigate(`/plugins`)}
                >
                    <ChevronLeft width="20px" height="20px" color="#F0F4FC" />
                    Back to All Plugins
                </Button>

                {plugin && pluginId && (
                    <>
                        <section className="plugin-header">
                            <img src={logo} alt="" />
                            <section className="plugin-details">
                                <h2 className="plugin-title">{plugin.title}</h2>
                                <p className="plugin-description">{plugin.description}</p>
                                <section className="plugin-installaion">
                                    {isConnected ? (
                                        isInstalled ? (
                                            <Button
                                                size="small"
                                                type="button"
                                                styleType="danger"
                                                onClick={() => setUninstallModalOpen(true)}
                                            >
                                                Uninstall
                                            </Button>
                                        ) : (
                                            <Button
                                                size="small"
                                                type="button"
                                                styleType="primary"
                                                onClick={handleStartReshare}
                                            >
                                                Install
                                            </Button>
                                        )
                                    ) : (
                                        <Button
                                            size="small"
                                            type="button"
                                            styleType="primary"
                                            onClick={async () => connect()}
                                        >
                                            Connect
                                        </Button>
                                    )}
                                    {isInstalled && (
                                        <Button
                                            size="small"
                                            type="button"
                                            styleType="secondary"
                                            onClick={() => setShowRecipeSchema(true)}
                                            style={{ marginLeft: 8 }}
                                        >
                                            View Policy Schema
                                        </Button>
                                    )}
                                    <div className="pricingInfo">
                                        {(!plugin?.pricing || plugin.pricing.length === 0) && (
                                            <aside>This plugin is free</aside>
                                        )}
                                        {plugin?.pricing?.map((price: PluginPricing) => (
                                            <div key={price.id}>{pricingText(price)}</div>
                                        ))}
                                    </div>
                                </section>
                            </section>
                        </section>

                        {isInstalled && <PolicyTable />}

                        {showRecipeSchema && (
                            <RecipeSchema
                                plugin={plugin}
                                onClose={() => {
                                    // TODO: Refetch Policies
                                    setShowRecipeSchema(false);
                                }}
                            />
                        )}

                        <ReviewProvider pluginId={plugin.id} ratings={plugin.ratings}>
                            <Reviews />
                        </ReviewProvider>

                        <Modal isOpen={uninstallModalOpen} onClose={() => setUninstallModalOpen(false)} variant="modal">
                            <>
                                <h4 className="">{`Are you sure you want unistall this plugin?`}</h4>
                                <div className="modal-actions">
                                    <Button
                                        ariaLabel="Delete policy"
                                        className="button secondary medium"
                                        type="button"
                                        styleType="tertiary"
                                        size="small"
                                        onClick={() => setUninstallModalOpen(false)}
                                    >
                                        Cancel
                                    </Button>
                                    <Button size="small" type="button" styleType="danger" onClick={uninstallPlugin}>
                                        Confirm
                                    </Button>
                                </div>
                            </>
                        </Modal>
                    </>
                )}
            </div>
        </>
    );
};

export default PluginDetail;
