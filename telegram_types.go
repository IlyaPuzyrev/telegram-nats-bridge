package main

// User represents a Telegram user or bot
type User struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name,omitempty"`
	Username                string `json:"username,omitempty"`
	LanguageCode            string `json:"language_code,omitempty"`
	IsPremium               bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu   bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`
	CanConnectToBusiness    bool   `json:"can_connect_to_business,omitempty"`
}

// Chat represents a chat
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // “private”, “group”, “supergroup” or “channel”
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	IsForum   bool   `json:"is_forum,omitempty"`
}

// Message represents a message
type Message struct {
	MessageID                     int64           `json:"message_id"`
	MessageThreadID               int64           `json:"message_thread_id,omitempty"`
	From                          *User           `json:"from,omitempty"`
	SenderChat                    *Chat           `json:"sender_chat,omitempty"`
	SenderBoostCount              int             `json:"sender_boost_count,omitempty"`
	SenderBusinessBot             *User           `json:"sender_business_bot,omitempty"`
	Date                          int64           `json:"date"`
	BusinessConnectionID          string          `json:"business_connection_id,omitempty"`
	Chat                          *Chat           `json:"chat"`
	ForwardOrigin                 interface{}     `json:"forward_origin,omitempty"`
	IsTopicMessage                bool            `json:"is_topic_message,omitempty"`
	IsAutomaticForward            bool            `json:"is_automatic_forward,omitempty"`
	ReplyToMessage                *Message        `json:"reply_to_message,omitempty"`
	ExternalReply                 interface{}     `json:"external_reply,omitempty"`
	Quote                         interface{}     `json:"quote,omitempty"`
	ReplyToStory                  interface{}     `json:"reply_to_story,omitempty"`
	ViaBot                        *User           `json:"via_bot,omitempty"`
	EditDate                      int64           `json:"edit_date,omitempty"`
	HasProtectedContent           bool            `json:"has_protected_content,omitempty"`
	IsFromOffline                 bool            `json:"is_from_offline,omitempty"`
	MediaGroupID                  string          `json:"media_group_id,omitempty"`
	AuthorSignature               string          `json:"author_signature,omitempty"`
	Text                          string          `json:"text,omitempty"`
	Entities                      []MessageEntity `json:"entities,omitempty"`
	LinkPreviewOptions            interface{}     `json:"link_preview_options,omitempty"`
	EffectID                      string          `json:"effect_id,omitempty"`
	Animation                     interface{}     `json:"animation,omitempty"`
	Audio                         interface{}     `json:"audio,omitempty"`
	Document                      interface{}     `json:"document,omitempty"`
	PaidMedia                     interface{}     `json:"paid_media,omitempty"`
	Photo                         []PhotoSize     `json:"photo,omitempty"`
	Sticker                       interface{}     `json:"sticker,omitempty"`
	Video                         interface{}     `json:"video,omitempty"`
	VideoNote                     interface{}     `json:"video_note,omitempty"`
	Voice                         interface{}     `json:"voice,omitempty"`
	Caption                       string          `json:"caption,omitempty"`
	CaptionEntities               []MessageEntity `json:"caption_entities,omitempty"`
	ShowCaptionAboveMedia         bool            `json:"show_caption_above_media,omitempty"`
	HasMediaSpoiler               bool            `json:"has_media_spoiler,omitempty"`
	Contact                       interface{}     `json:"contact,omitempty"`
	Dice                          interface{}     `json:"dice,omitempty"`
	Game                          interface{}     `json:"game,omitempty"`
	Poll                          interface{}     `json:"poll,omitempty"`
	Venue                         interface{}     `json:"venue,omitempty"`
	Location                      interface{}     `json:"location,omitempty"`
	NewChatMembers                []User          `json:"new_chat_members,omitempty"`
	LeftChatMember                *User           `json:"left_chat_member,omitempty"`
	NewChatTitle                  string          `json:"new_chat_title,omitempty"`
	NewChatPhoto                  []PhotoSize     `json:"new_chat_photo,omitempty"`
	DeleteChatPhoto               bool            `json:"delete_chat_photo,omitempty"`
	GroupChatCreated              bool            `json:"group_chat_created,omitempty"`
	SupergroupChatCreated         bool            `json:"supergroup_chat_created,omitempty"`
	ChannelChatCreated            bool            `json:"channel_chat_created,omitempty"`
	MessageAutoDeleteTimerChanged interface{}     `json:"message_auto_delete_timer_changed,omitempty"`
	MigrateToChatID               int64           `json:"migrate_to_chat_id,omitempty"`
	MigrateFromChatID             int64           `json:"migrate_from_chat_id,omitempty"`
	PinnedMessage                 interface{}     `json:"pinned_message,omitempty"`
	Invoice                       interface{}     `json:"invoice,omitempty"`
	SuccessfulPayment             interface{}     `json:"successful_payment,omitempty"`
	RefundedPayment               interface{}     `json:"refunded_payment,omitempty"`
	UsersShared                   interface{}     `json:"users_shared,omitempty"`
	ChatShared                    interface{}     `json:"chat_shared,omitempty"`
	ConnectedWebsite              string          `json:"connected_website,omitempty"`
	WriteAccessAllowed            interface{}     `json:"write_access_allowed,omitempty"`
	PassportData                  interface{}     `json:"passport_data,omitempty"`
	ProximityAlertTriggered       interface{}     `json:"proximity_alert_triggered,omitempty"`
	BoostAdded                    interface{}     `json:"boost_added,omitempty"`
	ChatBackgroundSet             interface{}     `json:"chat_background_set,omitempty"`
	ForumTopicCreated             interface{}     `json:"forum_topic_created,omitempty"`
	ForumTopicEdited              interface{}     `json:"forum_topic_edited,omitempty"`
	ForumTopicClosed              interface{}     `json:"forum_topic_closed,omitempty"`
	ForumTopicReopened            interface{}     `json:"forum_topic_reopened,omitempty"`
	GeneralForumTopicHidden       interface{}     `json:"general_forum_topic_hidden,omitempty"`
	GeneralForumTopicUnhidden     interface{}     `json:"general_forum_topic_unhidden,omitempty"`
	GiveawayCreated               interface{}     `json:"giveaway_created,omitempty"`
	Giveaway                      interface{}     `json:"giveaway,omitempty"`
	GiveawayWinners               interface{}     `json:"giveaway_winners,omitempty"`
	GiveawayCompleted             interface{}     `json:"giveaway_completed,omitempty"`
	VideoChatScheduled            interface{}     `json:"video_chat_scheduled,omitempty"`
	VideoChatStarted              interface{}     `json:"video_chat_started,omitempty"`
	VideoChatEnded                interface{}     `json:"video_chat_ended,omitempty"`
	VideoChatParticipantsInvited  interface{}     `json:"video_chat_participants_invited,omitempty"`
	WebAppData                    interface{}     `json:"web_app_data,omitempty"`
	ReplyMarkup                   interface{}     `json:"reply_markup,omitempty"`
}

// MessageEntity represents one special entity in a text message
type MessageEntity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	User          *User  `json:"user,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// PhotoSize represents one size of a photo or a file/sticker thumbnail
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// Update represents an incoming update from Telegram
type Update struct {
	UpdateID                int64       `json:"update_id"`
	Message                 *Message    `json:"message,omitempty"`
	EditedMessage           *Message    `json:"edited_message,omitempty"`
	ChannelPost             *Message    `json:"channel_post,omitempty"`
	EditedChannelPost       *Message    `json:"edited_channel_post,omitempty"`
	BusinessConnection      interface{} `json:"business_connection,omitempty"`
	BusinessMessage         *Message    `json:"business_message,omitempty"`
	EditedBusinessMessage   *Message    `json:"edited_business_message,omitempty"`
	DeletedBusinessMessages interface{} `json:"deleted_business_messages,omitempty"`
	MessageReaction         interface{} `json:"message_reaction,omitempty"`
	MessageReactionCount    interface{} `json:"message_reaction_count,omitempty"`
	InlineQuery             interface{} `json:"inline_query,omitempty"`
	ChosenInlineResult      interface{} `json:"chosen_inline_result,omitempty"`
	CallbackQuery           interface{} `json:"callback_query,omitempty"`
	ShippingQuery           interface{} `json:"shipping_query,omitempty"`
	PreCheckoutQuery        interface{} `json:"pre_checkout_query,omitempty"`
	PurchasedPaidMedia      interface{} `json:"purchased_paid_media,omitempty"`
	Poll                    interface{} `json:"poll,omitempty"`
	PollAnswer              interface{} `json:"poll_answer,omitempty"`
	MyChatMember            interface{} `json:"my_chat_member,omitempty"`
	ChatMember              interface{} `json:"chat_member,omitempty"`
	ChatJoinRequest         interface{} `json:"chat_join_request,omitempty"`
	ChatBoost               interface{} `json:"chat_boost,omitempty"`
	RemovedChatBoost        interface{} `json:"removed_chat_boost,omitempty"`
}

// GetUpdatesResponse represents the response from getUpdates method
type GetUpdatesResponse struct {
	Ok          bool     `json:"ok"`
	Result      []Update `json:"result,omitempty"`
	ErrorCode   int      `json:"error_code,omitempty"`
	Description string   `json:"description,omitempty"`
}
