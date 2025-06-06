import { useEffect, useRef, useState } from 'react';
import ULIComponent from './ULIComponent';
import {
    CheckCircleIcon,
    ChevronDownIcon,
    ClockIcon,
    PauseCircleIcon,
    PresentationChartLineIcon,
    XCircleIcon
} from '@heroicons/react/24/outline';
import ModifyClassModal from './modals/ModifyClassModal';
import { showModal } from './modals';
import {
    Class,
    ClassStatusMap,
    ClassStatusOptions,
    SelectedClassStatus,
    ServerResponseMany,
    ServerResponseOne
} from '@/common';
import { KeyedMutator } from 'swr';

export function isArchived(program_class: Class): boolean {
    return !(
        program_class.archived_at === null ||
        program_class.archived_at === '0001-01-01T00:00:00Z'
    );
}

function isCompletedOrCancelled(program_class: Class): boolean {
    return (
        program_class.status === SelectedClassStatus.Completed ||
        program_class.status === SelectedClassStatus.Cancelled
    );
}

function SelectedClassStatusPill({
    closed,
    status
}: {
    closed: boolean;
    status: SelectedClassStatus;
}) {
    let icon, background;

    switch (status) {
        case SelectedClassStatus.Completed:
            icon = CheckCircleIcon;
            background = 'bg-[#DDFFCD] text-[#408D1C]';
            break;
        case SelectedClassStatus.Cancelled:
            icon = XCircleIcon;
            background = 'bg-[#FFDFDF] text-[#CA0000]';
            break;
        case SelectedClassStatus.Paused:
            icon = PauseCircleIcon;
            background = 'bg-grey-2 text-body-text';
            break;
        case SelectedClassStatus.Scheduled:
            icon = ClockIcon;
            background = 'bg-[#FFF3D4] text-[#ECAA00]';
            break;
        case SelectedClassStatus.Active:
            icon = PresentationChartLineIcon;
            background = 'bg-[#B0DFDA] text-[#002E2A]';
    }

    if (!icon || !background) return;

    return (
        <div
            className={`inline-flex items-center gap-1 catalog-pill mx-0 w-full justify-between ${background} ${closed ? '' : 'cursor-pointer'}`}
        >
            <ULIComponent icon={icon} />
            <span>{status}</span>
            {closed ? <div></div> : <ULIComponent icon={ChevronDownIcon} />}
        </div>
    );
}

export default function ClassStatus({
    program_class,
    status,
    mutateClasses
}: {
    program_class: Class;
    status: SelectedClassStatus;
    mutateClasses:
        | KeyedMutator<ServerResponseOne<Class>>
        | KeyedMutator<ServerResponseMany<Class>>;
}) {
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const [selectedStatus, setSelectedStatus] = useState(status);
    const modifyClassRef = useRef<HTMLDialogElement>(null);
    const [selectedModifyOption, setSelectedModifyOption] =
        useState<ClassStatusOptions>();

    const classStatusTransitions = new Map<
        SelectedClassStatus,
        ClassStatusOptions[]
    >([
        [
            SelectedClassStatus.Scheduled,
            [ClassStatusOptions.Active, ClassStatusOptions.Cancel]
        ],
        [
            SelectedClassStatus.Active,
            [
                ClassStatusOptions.Complete,
                ClassStatusOptions.Cancel,
                ClassStatusOptions.Pause
            ]
        ],
        [
            SelectedClassStatus.Paused,
            [ClassStatusOptions.Active, ClassStatusOptions.Cancel]
        ],
        [SelectedClassStatus.Completed, []], //just in case, setting to empty array
        [SelectedClassStatus.Cancelled, []]
    ]);

    function openSelectionModal(
        e: React.MouseEvent<HTMLDivElement, MouseEvent>,
        option: ClassStatusOptions
    ) {
        if (isArchived(program_class) || isCompletedOrCancelled(program_class))
            return;
        setDropdownOpen(false);
        setSelectedModifyOption(option);
        e.stopPropagation();
    }

    useEffect(() => {
        showModal(modifyClassRef);
    }, [selectedModifyOption]);

    return (
        <>
            <div
                className="relative"
                onClick={(e) => {
                    if (
                        isArchived(program_class) ||
                        isCompletedOrCancelled(program_class)
                    )
                        return;
                    setDropdownOpen(!dropdownOpen);
                    e.stopPropagation();
                }}
                onBlur={(e) => {
                    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
                        setDropdownOpen(false);
                    }
                }}
                tabIndex={0}
            >
                <SelectedClassStatusPill
                    closed={
                        isArchived(program_class) ||
                        isCompletedOrCancelled(program_class)
                    }
                    status={selectedStatus}
                />
                {dropdownOpen && (
                    <ul
                        className="absolute left-0 bg-inner-background rounded-box shadow-lg p-2 overflow-y-auto z-10 w-full"
                        tabIndex={0}
                    >
                        {classStatusTransitions
                            .get(selectedStatus)
                            ?.map((option) => {
                                if (ClassStatusMap[option] === selectedStatus)
                                    return null;

                                return (
                                    <li key={option} className="w-full">
                                        <div
                                            className="flex items-center space-x-2 px-2 py-1 hover:bg-grey-2 rounded cursor-pointer"
                                            onClick={(e) =>
                                                openSelectionModal(e, option)
                                            }
                                        >
                                            <span className="text-sm">
                                                {option}
                                            </span>
                                        </div>
                                    </li>
                                );
                            })}
                    </ul>
                )}
            </div>
            <ModifyClassModal
                ref={modifyClassRef}
                action={selectedModifyOption}
                program_class={program_class}
                mutate={mutateClasses}
                setSelectedStatus={setSelectedStatus}
                onClose={() => setSelectedModifyOption(undefined)}
            />
        </>
    );
}
