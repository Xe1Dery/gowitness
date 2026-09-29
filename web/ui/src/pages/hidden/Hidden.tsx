import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { EyeOffIcon, LoaderIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/hooks/use-toast";
import { WideSkeleton } from "@/components/loading";
import * as api from "@/lib/api/api";
import * as apitypes from "@/lib/api/types";
import { formatDistanceToNow } from 'date-fns';

const HiddenGroupsPage = () => {
  const [hidden, setHidden] = useState<apitypes.hiddengroups>();
  const [loading, setLoading] = useState(true);
  const [busyGroupId, setBusyGroupId] = useState<number | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      setHidden(await api.get('hiddengroups'));
    } catch (err) {
      toast({
        title: "API Error",
        variant: "destructive",
        description: `Failed to get hidden groups: ${err}`
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleUnhide = async (groupId: number) => {
    if (busyGroupId !== null) return;
    setBusyGroupId(groupId);
    try {
      await api.post('hiddengroupstoggle', { perception_hash_group_id: groupId });
      await fetchData();
    } catch (err) {
      toast({
        title: "API Error",
        variant: "destructive",
        description: `Failed to unhide group: ${err}`
      });
    } finally {
      setBusyGroupId(null);
    }
  };

  if (loading) return <WideSkeleton />;

  return (
    <div className="space-y-6">
      <div className="flex items-center space-x-2">
        <EyeOffIcon className="h-5 w-5" />
        <h2 className="text-xl font-semibold">Hidden Similar Groups</h2>
        {hidden && hidden.total > 0 ? (
          <Badge variant="secondary">{hidden.total}</Badge>
        ) : null}
      </div>

      {hidden && hidden.total > 0 ? (
        <div className="grid gap-4">
          {hidden.groups.map((g) => {
            const hiddenDate = new Date(g.hidden_at);
            return (
              <Card key={g.perception_hash_group_id}>
                <CardContent className="p-4 flex flex-wrap items-center justify-between gap-4">
                  <div className="space-y-1">
                    <div className="flex items-center space-x-2">
                      <span className="font-medium">Group #{g.perception_hash_group_id}</span>
                      <Badge variant="outline">{g.count} screenshot{g.count === 1 ? "" : "s"} hidden</Badge>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Hidden {formatDistanceToNow(hiddenDate, { addSuffix: true })} ({hiddenDate.toLocaleString()})
                      {g.notes ? <span className="ml-2">{g.notes}</span> : null}
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {g.hidden_by_result_id > 0 ? (
                      <Link to={`/screenshot/${g.hidden_by_result_id}`}>
                        <Button variant="outline" size="sm">View Source</Button>
                      </Link>
                    ) : null}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleUnhide(g.perception_hash_group_id)}
                      disabled={busyGroupId === g.perception_hash_group_id}
                    >
                      {busyGroupId === g.perception_hash_group_id ? (
                        <LoaderIcon className="mr-2 h-4 w-4 animate-spin" />
                      ) : null}
                      Unhide
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      ) : (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            No hidden groups. Open a screenshot and use &quot;Hide Similar&quot; to hide its visually similar group from results.
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default HiddenGroupsPage;
