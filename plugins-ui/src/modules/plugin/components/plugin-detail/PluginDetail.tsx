import Button from "@/modules/core/components/ui/button/Button";
import { useNavigate, useParams } from "react-router-dom";
import ChevronLeft from "@/assets/ChevronLeft.svg?react";
import logo from "../../../../assets/DCA-image.png"; // todo hardcoded until this image is stored in DB
import "./PluginDetail.css";
import { useEffect, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { Plugin } from "../../models/plugin";
import Reviews from "@/modules/review/components/reviews/Reviews";
import { publish } from "@/utils/eventBus";
import { ReviewProvider } from "@/modules/review/context/ReviewProvider";
import VulticonnectWalletService from "@/modules/shared/wallet/vulticonnectWalletService";
import RecipeSchema from "@/modules/plugin/components/recipe_schema/recipe_Schema";
import { useWallet } from "@/modules/shared/wallet/WalletProvider";

const PluginDetail = () => {
  const navigate = useNavigate();
  const [plugin, setPlugin] = useState<Plugin | null>(null);
  const [isInstalled, setIsInstalled] = useState<boolean>(false);
  const [showRecipeSchema, setShowRecipeSchema] = useState(false);
  const { isConnected, connectWallet, chain, publicKey } = useWallet();
  const { pluginId } = useParams<{ pluginId: string }>();

  const checkPluginInstalled = async () => {
    if (isConnected && pluginId && publicKey) {
      await MarketplaceService.isPluginInstalled(pluginId, publicKey);

      setIsInstalled(true);
    } else {
      setIsInstalled(false);
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

  useEffect(() => {
    checkPluginInstalled();
  }, [isConnected, pluginId, publicKey]);

  useEffect(() => {
    fetchPlugin();
  }, [pluginId]);

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
                    isInstalled ? null : (
                      <Button
                        size="small"
                        type="button"
                        styleType="primary"
                        onClick={async () => {
                          try {
                            await VulticonnectWalletService.startReshareSession(
                              pluginId
                            );
                          } catch (err) {
                            console.error(
                              "Failed to start reshare session",
                              err
                            );
                          }
                        }}
                      >
                        Install
                      </Button>
                    )
                  ) : (
                    <Button
                      size="small"
                      type="button"
                      styleType="primary"
                      onClick={async () => connectWallet(chain)}
                    >
                      Connect
                    </Button>
                  )}
                  <Button
                    size="small"
                    type="button"
                    styleType="secondary"
                    onClick={() => setShowRecipeSchema(true)}
                    style={{ marginLeft: 8 }}
                  >
                    View Policy Schema
                  </Button>
                  <aside>Plugin fee: 0.1% per trade</aside>
                </section>
              </section>
            </section>

            {showRecipeSchema && (
              <RecipeSchema
                pluginId={plugin.id}
                onClose={() => setShowRecipeSchema(false)}
              />
            )}

            <ReviewProvider pluginId={plugin.id} ratings={plugin.ratings}>
              <Reviews plugin={plugin} />
            </ReviewProvider>
          </>
        )}
      </div>
    </>
  );
};

export default PluginDetail;
