import { ActivityHistoryResponse } from '@/common';

function ActivityHistoryRowCard({
    activity
}: {
    activity: ActivityHistoryResponse;
}) {
    const getProgramClassesHistoryEventText = () => {
        let text;
        switch (activity.field_name) {
            case 'capacity':
                text = `Capacity set to ${activity.new_value} by ${activity.admin_username}`;
                break;
            case 'class':
                text = `Class created by ${activity.admin_username}`;
                break;
            case 'program':
                text = `Program created by ${activity.admin_username}`;
                break;
            case 'name':
                text = `Name set to ${activity.new_value} by ${activity.admin_username}`;
                break;
            case 'instructor_name':
                text = `Instructor name set to ${activity.new_value} by ${activity.admin_username}`;
                break;
            case 'description':
                text = `Description set by ${activity.admin_username}`;
                break;
            case 'status':
                text = `Status set to ${activity.new_value} by ${activity.admin_username}`;
                break;
            case 'archived_at':
                text = `Archived by ${activity.admin_username}`;
                break;
            case 'credit_hours':
                text = `Credit hours set to ${activity.new_value} by ${activity.admin_username}`;
                break;
            case 'funding_type':
                text = `Funding type set to ${activity.new_value.replace(/_/g, ' ')} by ${activity.admin_username}`;
                break;
            case 'is_active':
                text = `Available set to ${activity.new_value} by ${activity.admin_username}`;
                break;
        }
        return text;
    };

    let introText;
    switch (activity.action) {
        case 'account_creation':
            introText = `Account created by ${activity.admin_username}`;
            break;
        case 'facility_transfer':
            introText = `Account assigned to ${activity.facility_name} by ${activity.admin_username}`;
            break;
        case 'set_password':
            introText = `New password set by ${activity.user_username}`;
            break;
        case 'reset_password':
            introText = `Password reset initiated by ${activity.admin_username}`;
            break;
        case 'progclass_history':
            introText = getProgramClassesHistoryEventText();
            break;
    }
    if (!introText) return;
    return (
        <p className="body">
            {introText} (
            {new Date(activity.created_at).toLocaleDateString('en-US')})
        </p>
    );
}

export default ActivityHistoryRowCard;
