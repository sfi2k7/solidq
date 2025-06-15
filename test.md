**Overall Goal:** Build an SMS messaging system that is flexible, compliant with consent rules, and supports a variety of message types while leveraging 
Twilio for sending.  The system needs to be designed with potential future provider changes in mind.

**1. Core Functionality (Sending & Receiving):**

*   **Twilio Integration:** Primarily uses Twilio for sending (but design for easy swapping later).  Uses a single Twilio account, messaging service, 
and a "bank" of sender phone numbers.
*   **Smart Sending:**  Messages are scheduled and delayed based on recipient’s timezone to adhere to a local time window (default 8 AM - 7 PM). An 
override is available.
*   **Link Tracking:**  Support URL shortening and click tracking (decision on Twilio vs. internal solution is pending).
*   **Comprehensive Logging:**  All messages and related data will be logged.

**2. Consent and Compliance (Critical):**

*   **Double Opt-in:** A double opt-in workflow is mandatory for new/renewed opt-in calls.
*   **Global & Type-Specific Consent:** Recipients must be globally opted-in *and* opted-in to the specific message type to receive texts.
*   **STOP Handling:**  The "STOP" keyword (and other reserved words) must be captured and processed to revoke all consent.
*   **OptIn Indicator:** Includes a mechanism (the "OptIn Indicator") to trigger the opt-in workflow for new recipients or when renewing consent.
*   **Metadata Logging:** Extensive metadata is logged during opt-in: Phone Number, Opt-in Source, IP Address (if applicable), and Timestamp.

**3. Messaging Types & Routing:**

*   **Multiple Message Types:** Supports BBBAccreditation, Marketing, ComplaintsReviews, and Quote message types.
*   **Message Type Activation:** Users can be updated to opt-in to new message types *without* sending a text.
*   **Routing based on Consent:** The system will only send messages if the recipient has opted-in globally and to the specific message type.

**4. Querying & Reporting:**

*   **Robust Querying:**  Ability to retrieve past SMS messages based on: Date Range, BBB, Sent From/Code, Sent To, Type, Batch/Tag.
*   **Conversation Retrieval:**  Ability to retrieve entire conversations (outbound and inbound).



**Key Design Considerations:**

*   **Future-Proofing:** Designed for easy switching of SMS providers.
*   **Consent-Driven:**  The system's functionality heavily revolves around adhering to consent rules and ensuring compliance.
*   **Data Logging:**  Extensive data logging for compliance, reporting, and troubleshooting.