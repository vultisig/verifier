package email

const (
	QueueName = "portal:email_queue"

	TypePortalProposal = "email:portal:proposal"
	TypePortalApproval = "email:portal:approval"
	TypePortalPublish  = "email:portal:publish"

	TemplatePortalProposal = "vultisig-portal-proposal"
	TemplatePortalApproval = "vultisig-portal-approval"
	TemplatePortalPublish  = "vultisig-portal-publish"

	MergeVarPluginID     = "PLUGIN_ID"
	MergeVarPluginTitle  = "PLUGIN_TITLE"
	MergeVarContactEmail = "CONTACT_EMAIL"
	MergeVarProposalURL  = "PROPOSAL_URL"
	MergeVarPluginURL    = "PLUGIN_URL"
)

type PortalProposalTask struct {
	PluginID           string   `json:"plugin_id"`
	Title              string   `json:"title"`
	ContactEmail       string   `json:"contact_email"`
	ProposalURL        string   `json:"proposal_url"`
	NotificationEmails []string `json:"notification_emails"`
}

type PortalApprovalTask struct {
	PluginID     string `json:"plugin_id"`
	Title        string `json:"title"`
	ContactEmail string `json:"contact_email"`
}

type PortalPublishTask struct {
	PluginID     string `json:"plugin_id"`
	Title        string `json:"title"`
	ContactEmail string `json:"contact_email"`
	PluginURL    string `json:"plugin_url"`
}
